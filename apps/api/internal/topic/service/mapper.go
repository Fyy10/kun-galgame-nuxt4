package service

import (
	"context"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/userclient"
)

func (s *PollService) buildPollResponse(ctx context.Context, poll *topicModel.TopicPoll, userID int, canModerate bool) (dto.TopicPollResponse, bool) {
	options, _ := s.pollRepo.FindOptionsByPollID(poll.ID)
	hasVoted, _ := s.pollRepo.HasUserVoted(poll.ID, userID)
	canView := canViewResults(poll, userID, canModerate, hasVoted)

	var userVotedOptionIDs map[int]bool
	if userID > 0 {
		votedIDs, _ := s.pollRepo.FindUserVoteOptionIDs(poll.ID, userID)
		userVotedOptionIDs = make(map[int]bool, len(votedIDs))
		for _, id := range votedIDs {
			userVotedOptionIDs[id] = true
		}
	}

	optionResponses := make([]dto.PollOptionResponse, len(options))
	for i, opt := range options {
		var voteCount *int
		if canView {
			vc := opt.VoteCount
			voteCount = &vc
		}
		optionResponses[i] = dto.PollOptionResponse{
			ID:        opt.ID,
			Text:      opt.Text,
			VoteCount: voteCount,
			IsVoted:   userVotedOptionIDs[opt.ID],
		}
	}

	var voters []dto.KunUser
	var votersCount int
	var totalVoteCount *int
	if canView {
		if !poll.IsAnonymous {
			voterIDs, _ := s.pollRepo.FindDistinctVoterIDs(poll.ID, 5)
			vmap := s.userClient.Hydrate(ctx, voterIDs)
			voters = make([]dto.KunUser, 0, len(voterIDs))
			for _, id := range voterIDs {
				u := vmap[id]
				if !userclient.IsRenderable(u) {
					continue
				}
				voters = append(voters, dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar})
			}
		}
		vc, _ := s.pollRepo.CountDistinctVoters(poll.ID)
		votersCount = vc
		tc, _ := s.pollRepo.CountTotalVotes(poll.ID)
		totalVoteCount = &tc
	}
	if voters == nil {
		voters = []dto.KunUser{}
	}

	creatorU, _, _ := s.userClient.User(ctx, poll.UserID)
	if creatorU.ID == 0 {
		creatorU = userclient.Placeholder(poll.UserID)
	}
	if !userclient.IsRenderable(creatorU) {
		return dto.TopicPollResponse{}, false
	}

	return dto.TopicPollResponse{
		ID: poll.ID, Title: poll.Title, Description: poll.Description,
		MinChoice: poll.MinChoice, MaxChoice: poll.MaxChoice,
		Deadline: poll.Deadline, Type: poll.Type, Status: poll.Status,
		ResultVisibility: poll.ResultVisibility,
		IsAnonymous:      poll.IsAnonymous, CanChangeVote: poll.CanChangeVote,
		TopicID: poll.TopicID, Created: poll.CreatedAt, Updated: poll.UpdatedAt,
		User:     dto.KunUser{ID: creatorU.ID, Name: creatorU.Name, Avatar: creatorU.Avatar},
		Options:  optionResponses,
		HasVoted: hasVoted, Voters: voters,
		VotersCount: votersCount, VoteCount: totalVoteCount,
	}, true
}

func canViewResults(poll *topicModel.TopicPoll, userID int, canModerate, hasVoted bool) bool {
	if userID == poll.UserID || canModerate {
		return true
	}
	isPollFinished := poll.Status == "closed" ||
		(poll.Deadline != nil && time.Now().After(*poll.Deadline))

	switch poll.ResultVisibility {
	case "always":
		return true
	case "after_vote":
		return hasVoted
	case "after_deadline":
		return isPollFinished
	default:
		return false
	}
}

func toTopicCard(r repository.TopicCardRow, sections []string, miniApps []string) dto.TopicCard {
	if sections == nil {
		sections = []string{}
	}
	covers := []string(r.CoverImages)
	if covers == nil {
		covers = []string{}
	}
	return dto.TopicCard{
		ID:             r.ID,
		Title:          r.Title,
		View:           r.View,
		Sections:       sections,
		CoverImages:    covers,
		CoverImageMeta: markdown.ResolveContentImageMeta(covers),
		User: dto.KunUser{
			ID:     r.UserID,
			Name:   r.UserName,
			Avatar: r.UserAvatar,
		},
		Status:           r.Status,
		HasBestAnswer:    r.BestAnswerID != nil,
		MiniApps:         miniApps,
		IsNSFW:           r.IsNSFW,
		LikeCount:        r.LikeCount,
		ReplyCount:       r.ReplyCount,
		CommentCount:     r.CommentCount,
		StatusUpdateTime: r.StatusUpdateTime,
		Created:          r.Created,
		UpvoteTime:       r.UpvoteTime,
	}
}
