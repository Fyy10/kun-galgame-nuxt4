package engagement_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/community/engagement"
	"kun-galgame-api/pkg/communityclient"
)

func serve(t *testing.T, states []map[string]any) (*engagement.Service, *[]string) {
	t.Helper()
	paths := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		var data any
		switch r.URL.Path {
		case "/threads/states":
			data = map[string]any{"states": states}
		default:
			data = map[string]any{
				"thread_id": 7, "user_id": 3, "last_read_post_number": 9,
				"highest_post_number": 9, "unread_count": 0, "notification_level": 3,
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "成功", "data": data})
	}))
	t.Cleanup(srv.Close)

	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	return engagement.New(cli, anchor.New(nil, nil)), &paths
}

// Reporting a receipt for a wall the reader merely opened would create the row
// at notification level normal, and the unread listing counts every level but
// muted — so every game page they ever glanced at would light the red dot.
func TestMarkReadDoesNotEnrolAStranger(t *testing.T) {
	svc, paths := serve(t, nil)

	state, appErr := svc.MarkRead(context.Background(), 3, 7)
	if appErr != nil {
		t.Fatalf("MarkRead: %v", appErr)
	}
	if slices.Contains(*paths, "POST /threads/7/read") {
		t.Errorf("reported a read receipt for a wall with no state row: %v", *paths)
	}
	if state.Subscribed {
		t.Errorf("state = %+v, want not subscribed", state)
	}
}

func TestMarkReadAdvancesASubscriber(t *testing.T) {
	svc, paths := serve(t, []map[string]any{
		{"thread_id": 7, "user_id": 3, "last_read_post_number": 4, "highest_post_number": 9,
			"unread_count": 5, "notification_level": 3},
	})

	state, appErr := svc.MarkRead(context.Background(), 3, 7)
	if appErr != nil {
		t.Fatalf("MarkRead: %v", appErr)
	}
	if !slices.Contains(*paths, "POST /threads/7/read") {
		t.Errorf("no read receipt sent for a subscriber: %v", *paths)
	}
	if !state.Subscribed || state.UnreadCount != 0 {
		t.Errorf("state = %+v, want subscribed and cleared", state)
	}
}

// The level upsert starts a fresh row at post 0, so following a wall without a
// receipt would hand the reader its whole backlog as unread.
func TestSetLevelClearsTheBacklogButMutingDoesNot(t *testing.T) {
	svc, paths := serve(t, nil)
	if _, appErr := svc.SetLevel(context.Background(), 3, 7, communityclient.NotificationWatching); appErr != nil {
		t.Fatalf("SetLevel: %v", appErr)
	}
	if !slices.Contains(*paths, "POST /threads/7/read") {
		t.Errorf("following a wall left its backlog unread: %v", *paths)
	}

	svc, paths = serve(t, nil)
	if _, appErr := svc.SetLevel(context.Background(), 3, 7, communityclient.NotificationMuted); appErr != nil {
		t.Fatalf("SetLevel: %v", appErr)
	}
	if slices.Contains(*paths, "POST /threads/7/read") {
		t.Errorf("muting sent a pointless read receipt: %v", *paths)
	}
}

func TestSetLevelRejectsAnUnknownLevel(t *testing.T) {
	svc, _ := serve(t, nil)
	if _, appErr := svc.SetLevel(context.Background(), 3, 7, 9); appErr == nil {
		t.Error("level 9 was accepted")
	}
}
