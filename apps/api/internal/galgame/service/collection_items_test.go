package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"kun-galgame-api/pkg/catalogclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func folderJSON(id int64, visibility string, itemCount int) string {
	return `{"object":"folder","id":"` + strconv.FormatInt(id, 10) +
		`","owner_uid":"90769","name":"","description":"","visibility":"` + visibility +
		`","is_default":true,"item_count":` + strconv.Itoa(itemCount) +
		`,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-09-20T00:00:00Z"}`
}

func itemJSON(workID int64, next string) string {
	cursor := "null"
	if next != "" {
		cursor = `"` + next + `"`
	}
	return `{"items":[{"object":"folder_item","folder_id":"4535","work_id":"` +
		strconv.FormatInt(workID, 10) +
		`","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}],"next_cursor":` + cursor + `}`
}

func TestLoadFolderContentsUsesThePublicLaneForAPublicFolder(t *testing.T) {
	var meItems int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/folders/4535" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(folderJSON(4535, "public", 2)))
		case r.URL.Path == "/v2/folders/4535/items":
			if r.URL.Query().Get("cursor") == "" {
				_, _ = w.Write([]byte(itemJSON(1, "cur_a")))
				return
			}
			_, _ = w.Write([]byte(itemJSON(2, "")))
		case strings.HasPrefix(r.URL.Path, "/v2/me/"):
			atomic.AddInt32(&meItems, 1)
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"detail":"Daily quota exceeded."}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	s := &CollectionService{catalog: catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"})}
	folder, items, err := s.loadFolderContents(context.Background(), 4535, true, "user-jwt")
	if err != nil {
		t.Fatalf("loadFolderContents: %v", err)
	}
	if folder == nil || folder.ID != 4535 || len(items) != 2 {
		t.Fatalf("folder %+v items %d", folder, len(items))
	}
	if atomic.LoadInt32(&meItems) != 0 {
		t.Fatalf("owner of a public folder still hit the user-plane %d times", meItems)
	}
}

func TestLoadFolderContentsCachesTheWalk(t *testing.T) {
	var itemHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/folders/4535":
			_, _ = w.Write([]byte(folderJSON(4535, "public", 1)))
		case r.URL.Path == "/v2/folders/4535/items":
			atomic.AddInt32(&itemHits, 1)
			_, _ = w.Write([]byte(itemJSON(9, "")))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	s := &CollectionService{
		catalog: catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"}),
		rdb:     redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}
	if _, items, err := s.loadFolderContents(context.Background(), 4535, false, ""); err != nil || len(items) != 1 {
		t.Fatalf("first load: items=%d err=%v", len(items), err)
	}
	if _, items, err := s.loadFolderContents(context.Background(), 4535, false, ""); err != nil || len(items) != 1 {
		t.Fatalf("second load: items=%d err=%v", len(items), err)
	}
	if got := atomic.LoadInt32(&itemHits); got != 1 {
		t.Fatalf("item walks = %d, want 1 (the second page view must be served from cache)", got)
	}
}

func TestLoadFolderContentsFallsBackToTheOwnerLaneForAPrivateFolder(t *testing.T) {
	var meHits, publicItems int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/folders/7":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND","detail":"No such folder."}`))
		case r.URL.Path == "/v2/folders/7/items":
			atomic.AddInt32(&publicItems, 1)
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == "/v2/me/folders/7":
			atomic.AddInt32(&meHits, 1)
			_, _ = w.Write([]byte(folderJSON(7, "private", 1)))
		case r.URL.Path == "/v2/me/folders/7/items":
			atomic.AddInt32(&meHits, 1)
			_, _ = w.Write([]byte(itemJSON(3, "")))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	s := &CollectionService{catalog: catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"})}
	folder, items, err := s.loadFolderContents(context.Background(), 7, true, "user-jwt")
	if err != nil {
		t.Fatalf("loadFolderContents: %v", err)
	}
	if folder == nil || folder.Visibility != "private" || len(items) != 1 || items[0].WorkID != 3 {
		t.Fatalf("folder %+v items %+v", folder, items)
	}
	if atomic.LoadInt32(&meHits) < 2 {
		t.Fatalf("private folder did not use the owner lane (%d me hits)", meHits)
	}
	if atomic.LoadInt32(&publicItems) != 0 {
		t.Fatalf("private folder leaked onto the public items lane")
	}
}

func TestListFoldersForViewerFallsBackToPublicOnQuota(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/me/folders":
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":"QUOTA_EXCEEDED","detail":"Daily quota exceeded."}`))
		case "/v2/folders":
			_, _ = w.Write([]byte(`{"items":[` + folderJSON(4535, "public", 1) + `],"next_cursor":null}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	s := &CollectionService{catalog: catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"})}
	folders, previewToken, err := s.listFoldersForViewer(context.Background(), 90769, 90769, "user-jwt")
	if err != nil {
		t.Fatalf("listFoldersForViewer: %v", err)
	}
	if len(folders) != 1 || folders[0].ID != 4535 {
		t.Fatalf("folders %+v", folders)
	}
	if previewToken != "" {
		t.Fatalf("preview after a quota fallback still carries the user token")
	}
}
