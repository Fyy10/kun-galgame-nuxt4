package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/userclient"
)

func reactionSummaries(
	samples []repository.ReactionSample,
	users map[int]userclient.User,
	cdn string,
	mine map[string]struct{},
	viewer *middleware.UserInfo,
) []ReactionSummary {
	type acc struct {
		sum     ReactionSummary
		firstAt int64
	}
	order := make([]string, 0)
	byTok := map[string]*acc{}
	for _, row := range samples {
		a, ok := byTok[row.Reaction]
		if !ok {
			sum := ReactionSummary{
				Reaction: ReactionToken(row.Reaction),
				Count:    row.Count,
				Reactors: []repr.UserRef{},
			}
			if viewer != nil {
				_, hit := mine[row.Reaction]
				sum.Viewer = &ReactionViewer{HasReacted: hit}
			}
			a = &acc{sum: sum, firstAt: row.FirstAt.UnixNano()}
			byTok[row.Reaction] = a
			order = append(order, row.Reaction)
		}
		if len(a.sum.Reactors) >= 3 {
			continue
		}
		u, ok := users[row.UserID]
		if !ok || !userclient.IsRenderable(u) {
			continue
		}
		a.sum.Reactors = append(a.sum.Reactors, repr.NewUserRef(cdn, u))
	}
	out := make([]ReactionSummary, 0, len(order))
	for _, tok := range order {
		out = append(out, byTok[tok].sum)
	}
	return out
}

func groupReactionSummaries(
	samples []repository.ReactionSample,
	users map[int]userclient.User,
	cdn string,
	mineByOwner map[int]map[string]struct{},
	viewer *middleware.UserInfo,
) map[int][]ReactionSummary {
	byOwner := map[int][]repository.ReactionSample{}
	for _, row := range samples {
		byOwner[row.OwnerID] = append(byOwner[row.OwnerID], row)
	}
	out := map[int][]ReactionSummary{}
	for id, rows := range byOwner {
		out[id] = reactionSummaries(rows, users, cdn, mineByOwner[id], viewer)
	}
	return out
}

func tokenSet(tokens []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, t := range tokens {
		out[t] = struct{}{}
	}
	return out
}

func hasToken(set map[string]struct{}, tok string) bool {
	_, ok := set[tok]
	return ok
}
