package apiv1

import (
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/userclient"
)

func TestReactionSummariesOrderCountAndBannedSkip(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Second)
	samples := []repository.ReactionSample{
		{OwnerID: 1, Reaction: "like", UserID: 2, Count: 4, FirstAt: t0},
		{OwnerID: 1, Reaction: "like", UserID: 3, Count: 4, FirstAt: t0},
		{OwnerID: 1, Reaction: "like", UserID: 4, Count: 4, FirstAt: t0},
		{OwnerID: 1, Reaction: "like", UserID: 5, Count: 4, FirstAt: t0},
		{OwnerID: 1, Reaction: "heart", UserID: 4, Count: 1, FirstAt: t1},
	}
	users := map[int]userclient.User{
		2: {ID: 2, Name: "banned", Status: 1},
		3: {ID: 3, Name: "alice", Status: 0},
		4: {ID: 4, Name: "bob", Status: 0},
		5: {ID: 5, Name: "carol", Status: 0},
	}
	viewer := &middleware.UserInfo{ID: 4}
	got := reactionSummaries(samples, users, "https://cdn.example", map[string]struct{}{"like": {}}, viewer)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Reaction != "like" || got[1].Reaction != "heart" {
		t.Fatalf("order %s then %s", got[0].Reaction, got[1].Reaction)
	}
	if got[0].Count != 4 || got[1].Count != 1 {
		t.Fatalf("counts %d %d", got[0].Count, got[1].Count)
	}
	if n := len(got[0].Reactors); n != 3 {
		t.Fatalf("like reactors %d, want 3 after skipping the banned first reactor", n)
	}
	if got[0].Reactors[0].ID != "3" || got[0].Reactors[1].ID != "4" || got[0].Reactors[2].ID != "5" {
		t.Fatalf("reactors %+v", got[0].Reactors)
	}
	if got[0].Viewer == nil || !got[0].Viewer.HasReacted {
		t.Fatal("viewer liked")
	}
	if got[1].Viewer == nil || got[1].Viewer.HasReacted {
		t.Fatal("viewer has not hearted")
	}
}

func TestReactionSummariesAnonymousViewerIsNull(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := reactionSummaries(
		[]repository.ReactionSample{{Reaction: "like", UserID: 3, Count: 1, FirstAt: t0}},
		map[int]userclient.User{3: {ID: 3, Name: "alice", Status: 0}},
		"", nil, nil,
	)
	if len(got) != 1 || got[0].Viewer != nil {
		t.Fatalf("%+v", got)
	}
	if len(got[0].Reactors) != 1 {
		t.Fatalf("reactors %d", len(got[0].Reactors))
	}
}

func TestReactionSummariesTieBreaksByToken(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := reactionSummaries(
		[]repository.ReactionSample{
			{Reaction: "clap", UserID: 1, Count: 1, FirstAt: t0},
			{Reaction: "heart", UserID: 1, Count: 1, FirstAt: t0},
		},
		map[int]userclient.User{1: {ID: 1, Name: "a", Status: 0}},
		"", nil, nil,
	)
	if len(got) != 2 || got[0].Reaction != "clap" || got[1].Reaction != "heart" {
		t.Fatalf("%+v", got)
	}
}

func TestReactionSummariesFewerThanThreeRenderable(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := reactionSummaries(
		[]repository.ReactionSample{
			{Reaction: "like", UserID: 2, Count: 3, FirstAt: t0},
			{Reaction: "like", UserID: 3, Count: 3, FirstAt: t0},
			{Reaction: "like", UserID: 4, Count: 3, FirstAt: t0},
		},
		map[int]userclient.User{
			2: {ID: 2, Status: 1},
			3: {ID: 3, Name: "alice", Status: 0},
			4: {ID: 4, Status: 1},
		},
		"", nil, nil,
	)
	if len(got) != 1 || got[0].Count != 3 || len(got[0].Reactors) != 1 || got[0].Reactors[0].ID != "3" {
		t.Fatalf("%+v", got)
	}
}
