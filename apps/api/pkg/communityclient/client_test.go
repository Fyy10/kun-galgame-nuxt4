package communityclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func newTestClient(baseURL string) *communityclient.Client {
	return communityclient.New(communityclient.Config{BaseURL: baseURL, ClientID: "cid", ClientSecret: "sec"})
}

func TestNotConfigured(t *testing.T) {
	c := communityclient.New(communityclient.Config{BaseURL: "http://x"})
	if c.Configured() {
		t.Fatal("Configured() true without creds")
	}
	if _, err := c.GetComments(context.Background(), communityclient.AnchorSiteGame, "1", "", ""); !errors.Is(err, communityclient.ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

func TestGetCommentsEnvelopeAndAuth(t *testing.T) {
	var gotAuth, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{
				"thread": map[string]any{"id": 7, "site": "kungal", "kind": 1, "anchor_kind": 1, "anchor_id": "42", "content_rating": 0, "status": 0, "posts_count": 2, "participants_count": 1, "highest_post_number": 2, "created_by": 1, "created_at": "2026-07-13T00:00:00Z"},
				"posts": []any{
					map[string]any{"id": 100, "thread_id": 7, "post_number": 1, "author_id": 1, "content_raw": "hi", "content_html": "<p>hi</p>", "content_rating": 0, "status": 0, "created_at": "2026-07-13T00:00:00Z"},
				},
				"next_cursor": "",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).GetComments(context.Background(), communityclient.AnchorSiteGame, "42", "", "")
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if gotAuth != "Basic Y2lkOnNlYw==" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/comments" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "anchor_kind=1") || !strings.Contains(gotQuery, "anchor_id=42") {
		t.Errorf("query %q missing the anchor", gotQuery)
	}
	if out.Thread == nil || out.Thread.ID != 7 || len(out.Posts) != 1 || out.Posts[0].ID != 100 {
		t.Errorf("decoded = %+v", out)
	}
}

// An anchor nobody has commented on answers with no thread at all — the page
// must render empty rather than dereference it.
func TestGetCommentsUncommentedAnchor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功", "data": map[string]any{"posts": []any{}},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).GetComments(context.Background(), communityclient.AnchorSiteGame, "42", "", "")
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if out.Thread != nil || len(out.Posts) != 0 {
		t.Errorf("decoded = %+v, want no thread and no posts", out)
	}
}

func TestCommentOnAnchor(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{
				"thread": map[string]any{"id": 7, "site": "kungal", "kind": 1, "anchor_kind": 1, "anchor_id": "42", "posts_count": 1},
				"post":   map[string]any{"id": 100, "thread_id": 7, "post_number": 1, "author_id": 3, "content_raw": "hi"},
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).CommentOnAnchor(context.Background(), communityclient.CommentRequest{
		AnchorKind: communityclient.AnchorSiteGame, AnchorID: "42", AuthorID: 3, Body: "hi",
	})
	if err != nil {
		t.Fatalf("CommentOnAnchor: %v", err)
	}
	if gotPath != "/comments" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"anchor_id":"42"`) || !strings.Contains(gotBody, `"author_id":3`) {
		t.Errorf("body %q missing the anchor or author", gotBody)
	}
	if out.Thread.ID != 7 || out.Post.ID != 100 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   map[string]any
		want   error
	}{
		{"forbidden site binding", http.StatusForbidden, nil, communityclient.ErrForbidden},
		{"tl0 rate limit", http.StatusTooManyRequests, nil, communityclient.ErrRateLimited},
		{"business code", http.StatusOK, map[string]any{"code": 40001, "message": "bad"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				if tc.body != nil {
					_ = json.NewEncoder(w).Encode(tc.body)
				}
			}))
			defer srv.Close()

			_, err := newTestClient(srv.URL).ListPosts(context.Background(), 7, "", "")
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Errorf("err = %v, want %v", err, tc.want)
				}
				return
			}
			var apiErr *communityclient.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != 40001 {
				t.Errorf("err = %v, want *APIError code=40001", err)
			}
		})
	}
}

func TestToggleReactionResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posts/100/reaction" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"added": true, "author_id": 55, "thread_id": 7, "anchor_kind": 1, "anchor_id": "42"},
		})
	}))
	defer srv.Close()

	res, err := newTestClient(srv.URL).ToggleReaction(context.Background(), 100, communityclient.ReactionToggleRequest{UserID: 9, Kind: communityclient.ReactionLike})
	if err != nil {
		t.Fatalf("ToggleReaction: %v", err)
	}
	if !res.Added || res.AuthorID != 55 || res.AnchorID != "42" {
		t.Errorf("result = %+v", res)
	}
}

func TestAuthorPosts(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"posts": []any{
					map[string]any{
						"post":   map[string]any{"id": 100, "thread_id": 7, "post_number": 3, "author_id": 55, "content_raw": "hi", "status": 0, "created_at": "2026-07-16T00:00:00Z"},
						"thread": map[string]any{"thread_id": 7, "title": "", "anchor_kind": 1, "anchor_id": "42"},
					},
				},
				"next_cursor": "99",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).AuthorPosts(context.Background(), 55, "150", 20, communityclient.AnchorSiteGame)
	if err != nil {
		t.Fatalf("AuthorPosts: %v", err)
	}
	if gotPath != "/authors/55/posts" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"after=150", "limit=20", "anchor_kind=1"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if len(out.Posts) != 1 || out.Posts[0].Post.ID != 100 || out.Posts[0].Thread.AnchorID != "42" || out.NextCursor != "99" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestAuthorStats(t *testing.T) {
	var gotQuery string
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit, gotQuery = true, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"stats": []any{
				map[string]any{"author_id": 55, "visible_posts": 9},
				map[string]any{"author_id": 56, "visible_posts": 0},
			}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	out, err := c.AuthorStats(context.Background(), []int64{55, 56})
	if err != nil {
		t.Fatalf("AuthorStats: %v", err)
	}
	if !strings.Contains(gotQuery, "ids=55%2C56") {
		t.Errorf("query %q missing joined ids", gotQuery)
	}
	if len(out.Stats) != 2 || out.Stats[0].VisiblePosts != 9 {
		t.Errorf("decoded = %+v", out)
	}
	hit = false
	if res, err := c.AuthorStats(context.Background(), nil); err != nil || len(res.Stats) != 0 || hit {
		t.Errorf("empty AuthorStats hit=%v res=%+v err=%v", hit, res, err)
	}
}

func TestAuthorPurge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/authors/55/purge" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"posts_purged": 3, "reactions_deleted": 2},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).AuthorPurge(context.Background(), 55)
	if err != nil {
		t.Fatalf("AuthorPurge: %v", err)
	}
	if out.PostsPurged != 3 || out.ReactionsDeleted != 2 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestResolvePosts(t *testing.T) {
	var gotPath, gotBody string
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit, gotPath = true, r.URL.Path
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"posts": []any{
				map[string]any{
					"post":   map[string]any{"id": 100, "thread_id": 7, "author_id": 55, "content_raw": "hi", "status": 0},
					"thread": map[string]any{"thread_id": 7, "anchor_kind": 1, "anchor_id": "42"},
				},
			}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	out, err := c.ResolvePosts(context.Background(), []int64{100, 200})
	if err != nil {
		t.Fatalf("ResolvePosts: %v", err)
	}
	if gotPath != "/posts/resolve" || !strings.Contains(gotBody, `"ids":[100,200]`) {
		t.Errorf("path = %q body = %q", gotPath, gotBody)
	}
	if len(out.Posts) != 1 || out.Posts[0].Post.ID != 100 {
		t.Errorf("decoded = %+v", out)
	}
	hit = false
	if res, err := c.ResolvePosts(context.Background(), nil); err != nil || len(res.Posts) != 0 || hit {
		t.Errorf("empty ResolvePosts hit=%v res=%+v err=%v", hit, res, err)
	}
}

func TestListPostsQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts": []any{}}})
	}))
	defer srv.Close()

	if _, err := newTestClient(srv.URL).ListPosts(context.Background(), 7, "50", "30"); err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	for _, want := range []string{"after=50", "limit=30"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}
