package apiv1

import (
	"context"
	"fmt"
	"slices"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func pollClosed() *problem.Problem {
	return problem.New(problem.CodePollClosed,
		"The poll no longer accepts votes: it is past closes_at. Nothing about the request is wrong.")
}

func voteAlreadyCast() *problem.Problem {
	return problem.New(problem.CodeVoteAlreadyCast,
		"The caller has already voted and this poll does not allow changing a vote.")
}

func pollChoiceCountField(poll *model.TopicPoll, n int) []problem.FieldError {
	switch {
	case n < poll.MinChoice:
		min := poll.MinChoice
		return []problem.FieldError{problem.AtPointer("/option_ids", problem.ReasonTooFewItems,
			"this poll asks for at least min_choice options", &problem.FieldParams{MinItems: &min})}
	case n > poll.MaxChoice:
		max := poll.MaxChoice
		return []problem.FieldError{problem.AtPointer("/option_ids", problem.ReasonTooManyItems,
			"this poll accepts at most max_choice options", &problem.FieldParams{MaxItems: &max})}
	default:
		return nil
	}
}

// Nothing tied option_id to poll_id and there is no composite foreign key, so
// hosting your own poll and naming another poll's option ids moved that
// option's vote_count. The id must be one of this poll's own options.
func (p *Polls) chosenOptionIDs(poll *model.TopicPoll, raw []repr.DecimalID) ([]int, *problem.Problem) {
	options, err := p.polls.FindOptionsByPollID(poll.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	belongs := make(map[int]struct{}, len(options))
	for _, opt := range options {
		belongs[opt.ID] = struct{}{}
	}
	out := make([]int, 0, len(raw))
	for i, value := range raw {
		id, ok := repr.ParseID(value)
		if !ok {
			return nil, validationFailed(pollUnknownOption(fmt.Sprintf("/option_ids/%d", i)))
		}
		if _, member := belongs[id]; !member {
			return nil, validationFailed(pollUnknownOption(fmt.Sprintf("/option_ids/%d", i)))
		}
		out = append(out, id)
	}
	if fields := pollChoiceCountField(poll, len(out)); fields != nil {
		return nil, validationFailed(fields...)
	}
	return out, nil
}

func sameChoice(a, b []int) bool {
	x := slices.Clone(a)
	y := slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}

func (p *Polls) setPollVote(ctx context.Context, in *setPollVoteInput) (*pollOutput, error) {
	poll, topic, user, prob := p.visiblePoll(ctx, in.PollID)
	if prob != nil {
		return nil, prob
	}
	chosen, prob := p.chosenOptionIDs(poll, in.Body.OptionIDs)
	if prob != nil {
		return nil, prob
	}
	err := p.db().Transaction(func(tx *gorm.DB) error {
		current, err := p.polls.LockUserVotes(tx, poll.ID, user.ID)
		if err != nil {
			return err
		}
		if sameChoice(current, chosen) {
			return nil
		}
		if prob := p.refuseVoteChange(poll, len(current) > 0); prob != nil {
			return txFail{prob}
		}
		return p.moveVotes(tx, poll.ID, user.ID, chosen)
	})
	if prob := txProblem(err); prob != nil {
		return nil, prob
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	return p.reloadOut(ctx, topic, poll.ID, user)
}

func (p *Polls) clearPollVote(ctx context.Context, in *pollInput) (*pollOutput, error) {
	poll, topic, user, prob := p.visiblePoll(ctx, in.PollID)
	if prob != nil {
		return nil, prob
	}
	err := p.db().Transaction(func(tx *gorm.DB) error {
		current, err := p.polls.LockUserVotes(tx, poll.ID, user.ID)
		if err != nil {
			return err
		}
		if len(current) == 0 {
			return nil
		}
		if prob := p.refuseVoteChange(poll, true); prob != nil {
			return txFail{prob}
		}
		return p.moveVotes(tx, poll.ID, user.ID, nil)
	})
	if prob := txProblem(err); prob != nil {
		return nil, prob
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	return p.reloadOut(ctx, topic, poll.ID, user)
}

func (p *Polls) refuseVoteChange(poll *model.TopicPoll, hasVoted bool) *problem.Problem {
	if pollIsClosed(poll, time.Now()) {
		return pollClosed()
	}
	if hasVoted && !poll.CanChangeVote {
		return voteAlreadyCast()
	}
	return nil
}

// Only rows that really moved move a counter. The delete names the options the
// caller is giving up and the insert conflicts on the ones they keep, so
// repeating the same PUT returns no rows either way and touches no count.
func (p *Polls) moveVotes(tx *gorm.DB, pollID, userID int, chosen []int) error {
	dropped, err := p.polls.DeleteUserVotesExcept(tx, pollID, userID, chosen)
	if err != nil {
		return err
	}
	if err := p.polls.AddOptionVoteCounts(tx, dropped, -1); err != nil {
		return err
	}
	var added []int
	for _, id := range chosen {
		_, inserted, err := p.polls.InsertUserVote(tx, pollID, id, userID)
		if err != nil {
			return err
		}
		if inserted {
			added = append(added, id)
		}
	}
	return p.polls.AddOptionVoteCounts(tx, added, 1)
}

func (p *Polls) listPollVotes(ctx context.Context, in *listPollVotesInput) (*listPollVotesOutput, error) {
	poll, _, user, prob := p.visiblePoll(ctx, in.PollID)
	if prob != nil {
		return nil, prob
	}
	// The legacy face answered 200 with an empty array here, so a caller could
	// not tell an empty poll from one they may not read.
	if poll.IsAnonymous {
		return nil, permissionRequired()
	}
	hasVoted := false
	if user != nil {
		choices, err := p.polls.ViewerChoices([]int{poll.ID}, user.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		hasVoted = len(choices) > 0
	}
	if !canViewPollResults(poll, user, hasVoted) {
		return nil, permissionRequired()
	}

	fp := historyFingerprint("poll_votes", poll.ID)
	pos, prob := decodeHistoryCursor(in.Cursor, fp)
	if prob != nil {
		return nil, prob
	}
	limit := historyLimit(in.Limit)
	rows, last, more, prob := p.gatherVotes(ctx, poll.ID, limit, pos)
	if prob != nil {
		return nil, prob
	}
	users, prob := p.reads.lookupUsers(ctx, userclient.CollectIDs(rows, func(r repository.PollVoteRow) int { return r.UserID }))
	if prob != nil {
		return nil, prob
	}
	items := make([]PollVote, 0, len(rows))
	for _, row := range rows {
		voter := repr.DeletedUserRef(row.UserID)
		if u, ok := users[row.UserID]; ok {
			voter = repr.NewUserRef(p.reads.cdn, u)
		}
		items = append(items, PollVote{
			Object:    "poll_vote",
			ID:        repr.ID(row.ID),
			PollID:    repr.ID(poll.ID),
			OptionID:  repr.ID(row.OptionID),
			Voter:     voter,
			CreatedAt: repr.Timestamp(row.Created),
		})
	}
	var next *string
	if more {
		cur := collect.EncodeCursor(historySort, fp, last.Created.UTC().Format(time.RFC3339Nano), fmt.Sprint(last.ID))
		next = &cur
	}
	return &listPollVotesOutput{Body: repr.NewList(items, next)}, nil
}

func (p *Polls) gatherVotes(
	ctx context.Context,
	pollID, limit int,
	pos *repository.EngageHistoryPos,
) ([]repository.PollVoteRow, repository.PollVoteRow, bool, *problem.Problem) {
	var (
		out  []repository.PollVoteRow
		last repository.PollVoteRow
	)
	window := pos
	for range maxWindows {
		rows, err := p.polls.ListPollVotesKeyset(pollID, limit, window)
		if err != nil {
			return nil, last, false, problem.Internal(err)
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		if len(rows) == 0 {
			return out, last, false, nil
		}
		voters, prob := p.reads.lookupUsers(ctx, userclient.CollectIDs(rows, func(r repository.PollVoteRow) int { return r.UserID }))
		if prob != nil {
			return nil, last, false, prob
		}
		for i, row := range rows {
			last = row
			if u, ok := voters[row.UserID]; ok && !userclient.IsRenderable(u) {
				continue
			}
			out = append(out, row)
			if len(out) == limit {
				return out, last, more || i < len(rows)-1, nil
			}
		}
		if !more {
			return out, last, false, nil
		}
		window = &repository.EngageHistoryPos{Created: last.Created, ID: last.ID}
	}
	return out, last, true, nil
}
