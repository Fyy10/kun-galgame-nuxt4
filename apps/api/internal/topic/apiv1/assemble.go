package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type replyExtras struct {
	comments map[int][]repository.CommentListRow
	samples  []repository.ReactionSample
	mine     map[int]map[string]struct{}
	likedC   map[int]bool
}

func (s *Service) loadReplyExtras(replies []model.TopicReply, viewer *middleware.UserInfo) (*replyExtras, *problem.Problem) {
	if s.comments == nil || s.replies == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	ids := make([]int, len(replies))
	for i, r := range replies {
		ids[i] = r.ID
	}
	rows, err := s.comments.ListByReplyIDs(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	byReply := map[int][]repository.CommentListRow{}
	commentIDs := make([]int, 0, len(rows))
	for _, c := range rows {
		byReply[c.TopicReplyID] = append(byReply[c.TopicReplyID], c)
		commentIDs = append(commentIDs, c.ID)
	}
	samples, err := s.replies.SampleReplyReactions(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	mine := map[int]map[string]struct{}{}
	likedC := map[int]bool{}
	if viewer != nil {
		raw, err := s.replies.GetUserRepliesReactions(ids, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		for id, toks := range raw {
			mine[id] = tokenSet(toks)
		}
		likedC, err = s.comments.FindCommentLikeStatus(viewer.ID, commentIDs)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	return &replyExtras{comments: byReply, samples: samples, mine: mine, likedC: likedC}, nil
}

func collectReplyUserIDs(replies []model.TopicReply, extra *replyExtras) []int {
	seen := map[int]struct{}{}
	var ids []int
	add := func(id int) {
		if id <= 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, r := range replies {
		add(r.UserID)
	}
	if extra == nil {
		return ids
	}
	for _, rows := range extra.comments {
		for _, c := range rows {
			add(c.UserID)
			add(c.TargetUserID)
		}
	}
	for _, row := range extra.samples {
		add(row.UserID)
	}
	return ids
}

func (s *Service) lookupUsers(ctx context.Context, ids []int) (map[int]userclient.User, *problem.Problem) {
	if s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if users == nil {
		users = map[int]userclient.User{}
	}
	return users, nil
}

func (s *Service) convertBodies(ctx context.Context, sources []string) ([]content.ContentDocument, *problem.Problem) {
	if s.convert == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	docs, err := s.convert.Convert(ctx, sources)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	return docs, nil
}

func (s *Service) mapReplies(
	topic *model.Topic,
	replies []model.TopicReply,
	docs []content.ContentDocument,
	extra *replyExtras,
	users map[int]userclient.User,
	moe map[int]int,
	viewer *middleware.UserInfo,
) []*Reply {
	react := groupReactionSummaries(extra.samples, users, s.cdn, extra.mine, viewer)
	pack := &replyPack{
		comments: extra.comments,
		react:    react,
		mine:     extra.mine,
		likedC:   extra.likedC,
		moe:      moe,
		users:    users,
	}
	out := make([]*Reply, len(replies))
	for i, row := range replies {
		doc := content.NewDocument(nil)
		if i < len(docs) {
			doc = docs[i]
		}
		out[i] = pack.mapOne(s.cdn, topic, row, doc, viewer)
	}
	return out
}

type replyPack struct {
	comments map[int][]repository.CommentListRow
	react    map[int][]ReactionSummary
	mine     map[int]map[string]struct{}
	likedC   map[int]bool
	moe      map[int]int
	users    map[int]userclient.User
}

func (p *replyPack) mapOne(cdn string, topic *model.Topic, row model.TopicReply, doc content.ContentDocument, viewer *middleware.UserInfo) *Reply {
	u, ok := p.users[row.UserID]
	if ok && !userclient.IsRenderable(u) {
		return nil
	}
	author := repr.DeletedUserRef(row.UserID)
	if ok {
		author = repr.NewUserRef(cdn, u)
	}
	mine := p.mine[row.ID]
	var rv *ReplyViewer
	if viewer != nil {
		rv = &ReplyViewer{HasLiked: hasToken(mine, "like"), HasDisliked: hasToken(mine, "dislike")}
	}
	react := p.react[row.ID]
	if react == nil {
		react = []ReactionSummary{}
	}
	return &Reply{
		Object:            "reply",
		ID:                repr.ID(row.ID),
		TopicID:           repr.ID(row.TopicID),
		Floor:             row.Floor,
		Author:            author,
		AuthorMoemoepoint: p.moe[row.UserID],
		Content:           doc,
		LikeCount:         row.LikeCount,
		DislikeCount:      row.DislikeCount,
		Reactions:         react,
		IsPinned:          topic != nil && topic.PinnedReplyID != nil && *topic.PinnedReplyID == row.ID,
		IsBestAnswer:      topic != nil && topic.BestAnswerID != nil && *topic.BestAnswerID == row.ID,
		Comments:          p.mapComments(cdn, row.ID, viewer),
		CreatedAt:         repr.Timestamp(row.CreatedAt),
		EditedAt:          repr.TimestampPtr(row.Edited),
		Viewer:            rv,
	}
}

func (p *replyPack) mapComments(cdn string, replyID int, viewer *middleware.UserInfo) []Comment {
	rows := p.comments[replyID]
	out := make([]Comment, 0, len(rows))
	for _, row := range rows {
		u, ok := p.users[row.UserID]
		if ok && !userclient.IsRenderable(u) {
			continue
		}
		author := repr.DeletedUserRef(row.UserID)
		if ok {
			author = repr.NewUserRef(cdn, u)
		}
		target := repr.DeletedUserRef(row.TargetUserID)
		if tu, tok := p.users[row.TargetUserID]; tok {
			target = repr.NewUserRef(cdn, tu)
		}
		var cv *CommentViewer
		if viewer != nil {
			cv = &CommentViewer{HasLiked: p.likedC[row.ID]}
		}
		out = append(out, Comment{
			Object:          "comment",
			ID:              repr.ID(row.ID),
			ReplyID:         repr.ID(row.TopicReplyID),
			ParentCommentID: optID(row.ParentCommentID),
			Author:          author,
			InReplyToUser:   target,
			Text:            row.Content,
			LikeCount:       row.LikeCount,
			CreatedAt:       repr.Timestamp(row.CreatedAt),
			EditedAt:        repr.TimestampPtr(row.Edited),
			Viewer:          cv,
		})
	}
	return out
}

func optID(p *int) *repr.DecimalID {
	if p == nil {
		return nil
	}
	id := repr.ID(*p)
	return &id
}

func replyBodies(rows []model.TopicReply) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Content
	}
	return out
}
