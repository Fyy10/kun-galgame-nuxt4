package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/catalogclient"
)

func TestGetMyInteractionsAsksHoldingsNotEveryItem(t *testing.T) {
	var holdings, items, folders int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/me/folders/holdings":
			atomic.AddInt32(&holdings, 1)
			if !strings.Contains(r.URL.RawQuery, "work_ids=") {
				t.Errorf("holdings query %q carries no work_ids", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"items":[{"object":"folder_holding","work_id":"42","folder_ids":["4535"]}],"next_cursor":null}`))
		case strings.HasSuffix(r.URL.Path, "/items"):
			atomic.AddInt32(&items, 1)
			w.WriteHeader(http.StatusInternalServerError)
		case r.URL.Path == "/v2/me/folders":
			atomic.AddInt32(&folders, 1)
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	s := &GalgameService{
		interactionRepo: &repository.GalgameInteractionRepository{},
		catalog:         catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"}),
	}
	got := s.GetMyInteractions(context.Background(), 0, "user-jwt", []int{42, 99})
	if len(got.Favorited) != 1 || got.Favorited[0] != 42 {
		t.Fatalf("favorited %v, want [42]", got.Favorited)
	}
	if atomic.LoadInt32(&holdings) != 1 {
		t.Fatalf("holdings calls = %d", holdings)
	}
	if atomic.LoadInt32(&items) != 0 || atomic.LoadInt32(&folders) != 0 {
		t.Fatalf("walked folders=%d items=%d, want neither", folders, items)
	}
}

func TestGetMyInteractionsWithoutWorkIDsDoesNotTouchCatalog(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		atomic.AddInt32(&hits, 1)
	}))
	t.Cleanup(srv.Close)

	s := &GalgameService{
		interactionRepo: &repository.GalgameInteractionRepository{},
		catalog:         catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"}),
	}
	got := s.GetMyInteractions(context.Background(), 0, "user-jwt", nil)
	if len(got.Favorited) != 0 {
		t.Fatalf("favorited %v, want none", got.Favorited)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("catalog was called %d times with no work ids", hits)
	}
}
