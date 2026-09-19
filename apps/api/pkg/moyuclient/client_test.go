package moyuclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(Config{BaseURL: srv.URL, APIKey: "nmk_live_test"})
}

func TestPatchesForWork(t *testing.T) {
	var gotReq *http.Request
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		_, _ = w.Write([]byte(`{"object":"list","next_cursor":null,"total":null,"items":[
			{"object":"patch","id":"61311","vndb_id":"v4145","catalog_work_id":"61311",
			 "web_url":"https://www.moyu.moe/galgame/61311","resources":[
				{"object":"patch_resource","id":"10463","patch_id":"61311","name":"汉化补丁",
				 "storage":"s3","size":"0.571 MB","model_name":"","note":"**注意**",
				 "type":["manual"],"language":["zh-Hans"],"platform":["windows"],
				 "download_count":12,"web_url":"https://www.moyu.moe/resource/10463",
				 "updated_at":"2026-08-29T02:51:07Z",
				 "publisher":{"object":"user","id":"2","name":"鲲","avatar_url":"https://img/a.webp"}}]}]}`))
	})

	patches, err := c.PatchesForWork(context.Background(), 61311)
	if err != nil {
		t.Fatalf("PatchesForWork: %v", err)
	}
	if gotReq.URL.Path != "/v2/moyu/patches" {
		t.Errorf("path = %q", gotReq.URL.Path)
	}
	if got := gotReq.Header.Get("Authorization"); got != "Bearer nmk_live_test" {
		t.Errorf("auth = %q", got)
	}
	q := gotReq.URL.Query()
	for param, want := range map[string]string{
		"refs":    "catalog:61311",
		"include": "resources,publisher",
		"nsfw":    "true",
	} {
		if got := q.Get(param); got != want {
			t.Errorf("%s = %q, want %q", param, got, want)
		}
	}

	if len(patches) != 1 || len(patches[0].Resources) != 1 {
		t.Fatalf("patches = %+v", patches)
	}
	r := patches[0].Resources[0]
	if r.ID != "10463" || r.WebURL != "https://www.moyu.moe/resource/10463" || r.DownloadCount != 12 {
		t.Errorf("resource = %+v", r)
	}
	if r.Publisher == nil || r.Publisher.ID != "2" || r.Publisher.AvatarURL != "https://img/a.webp" {
		t.Errorf("publisher = %+v", r.Publisher)
	}
}

func TestPatchesForWork_Problem(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":401,"code":"INVALID_CREDENTIAL","detail":"The application key is invalid, revoked or expired."}`))
	})

	_, err := c.PatchesForWork(context.Background(), 1)
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
}

func TestPatchesForWork_NotConfigured(t *testing.T) {
	c := New(Config{BaseURL: "https://api.nextmoe.dev"})
	if _, err := c.PatchesForWork(context.Background(), 1); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}
