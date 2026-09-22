package apiv1

import (
	"context"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/access"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type TopicState struct {
	Object         string          `json:"object" enum:"topic_state" maxLength:"11" doc:"Type discriminant. Always topic_state."`
	TopicID        repr.DecimalID  `json:"topic_id" doc:"Id of the topic this state is about."`
	HasFavorited   bool            `json:"has_favorited" doc:"Whether the caller favorited the topic."`
	ReactionTokens []ReactionToken `json:"reaction_tokens" maxItems:"64" doc:"The caller's own reaction tokens on the topic, oldest first. Empty array if none. Tokens, not tallies: reactions elsewhere in this API means the per-token summary with counts and reactors."`
}

type listTopicStatesInput struct {
	TopicIDs []repr.DecimalID `query:"topic_ids" required:"true" maxItems:"100" doc:"Topic ids to answer for, comma-separated. 1 to 100 of them."`
}

type listTopicStatesOutput struct {
	Body repr.BatchList[TopicState]
}

func (s *Service) listTopicStates(ctx context.Context, in *listTopicStatesInput) (*listTopicStatesOutput, error) {
	if s == nil || s.topics == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	requested, prob := parseTopicIDs(in.TopicIDs)
	if prob != nil {
		return nil, prob
	}

	unique := make([]int, 0, len(requested))
	seen := make(map[int]bool, len(requested))
	for _, id := range requested {
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}

	readable, prob := s.readableTopics(ctx, unique, user)
	if prob != nil {
		return nil, prob
	}
	favorited, err := s.topics.UserFavoritedTopicIDs(user.ID, unique)
	if err != nil {
		return nil, problem.Internal(err)
	}
	reactions, err := s.topics.UserTopicReactions(user.ID, unique)
	if err != nil {
		return nil, problem.Internal(err)
	}

	items := make([]TopicState, 0, len(unique))
	missing := []repr.DecimalID{}
	emitted := map[int]bool{}
	for _, id := range requested {
		if !readable[id] {
			if !emitted[id] {
				emitted[id] = true
				missing = append(missing, repr.ID(id))
			}
			continue
		}
		if emitted[id] {
			continue
		}
		emitted[id] = true
		items = append(items, TopicState{
			Object:         "topic_state",
			TopicID:        repr.ID(id),
			HasFavorited:   favorited[id],
			ReactionTokens: toReactionTokens(reactions[id]),
		})
	}
	return &listTopicStatesOutput{Body: repr.NewBatchList(items, missing)}, nil
}

func toReactionTokens(tokens []string) []ReactionToken {
	out := make([]ReactionToken, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, ReactionToken(t))
	}
	return out
}

// The schema already refuses an absent topic_ids, more than 100 of them, and
// anything that is not decimal digits. What it cannot say is that 0 is not an
// id, so that is all this checks.
func parseTopicIDs(parts []repr.DecimalID) ([]int, *problem.Problem) {
	out := make([]int, 0, len(parts))
	for i, part := range parts {
		id, ok := repr.ParseID(repr.DecimalID(strings.TrimSpace(string(part))))
		if !ok {
			return nil, validationFailed(problem.AtParameter("topic_ids", problem.ReasonInvalidFormat,
				"every id must be a positive decimal integer; item "+strconv.Itoa(i)+" is not", nil))
		}
		out = append(out, id)
	}
	return out, nil
}

// A topic the caller may not read is not distinguished from one that does not
// exist, so this answers only "which of these may they read".
func (s *Service) readableTopics(ctx context.Context, ids []int, user *middleware.UserInfo) (map[int]bool, *problem.Problem) {
	out := map[int]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.topics.FindByIDs(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	needGrants := make([]int, 0, len(rows))
	for i := range rows {
		if access.NeedsGrants(&rows[i]) {
			needGrants = append(needGrants, rows[i].ID)
		}
	}
	grants := map[int][]model.TopicAccessGrant{}
	if len(needGrants) > 0 {
		grants, err = s.topics.FindAccessGrantsForTopics(needGrants)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	authors := make([]int, 0, len(rows))
	for i := range rows {
		authors = append(authors, rows[i].UserID)
	}
	users, prob := s.lookupUsers(ctx, authors)
	if prob != nil {
		return nil, prob
	}
	for i := range rows {
		topic := &rows[i]
		if u, ok := users[topic.UserID]; ok && !userclient.IsRenderable(u) {
			continue
		}
		if !access.CanRead(topic, user, grants[topic.ID]) {
			continue
		}
		out[topic.ID] = true
	}
	return out, nil
}
