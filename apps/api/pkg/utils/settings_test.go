package utils

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// The cookie is written by the frontend's persisted-settings store, so it
// arrives URL-encoded and carries whatever keys that store happens to hold —
// including ones this API has never heard of.
func TestSettingsCookie(t *testing.T) {
	for _, tc := range []struct {
		why        string
		cookie     string
		wantSFW    bool
		wantOrigin bool
	}{
		{"no cookie at all", "", true, false},
		{"unparseable", "not json", true, false},
		{"a store that predates both keys", `{"showKUNGalgameRounded":"md"}`, true, false},
		{"nsfw on, names default", `{"showKUNGalgameContentLimit":"nsfw"}`, false, false},
		{"原名 on", `{"showKUNGalgamePreferOriginalName":true}`, true, true},
		{"both on", `{"showKUNGalgameContentLimit":"all","showKUNGalgamePreferOriginalName":true}`, false, true},
	} {
		app := fiber.New()
		var gotSFW, gotOrigin bool
		app.Get("/", func(c fiber.Ctx) error {
			gotSFW, gotOrigin = IsSFW(c), PrefersOriginalName(c)
			return nil
		})
		req := httptest.NewRequest("GET", "/", nil)
		if tc.cookie != "" {
			req.Header.Set("Cookie", "KUNGalgameSettings="+url.QueryEscape(tc.cookie))
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if gotSFW != tc.wantSFW || gotOrigin != tc.wantOrigin {
			t.Errorf("%s: sfw=%v origin=%v, want %v/%v",
				tc.why, gotSFW, gotOrigin, tc.wantSFW, tc.wantOrigin)
		}
	}
}

func TestNSFWHeaderBeatsCookie(t *testing.T) {
	for _, tc := range []struct {
		why     string
		header  string
		cookie  string
		wantSFW bool
	}{
		{"header on, no cookie", "1", "", false},
		{"header off overrides an nsfw cookie", "false", `{"showKUNGalgameContentLimit":"nsfw"}`, true},
		{"unparseable header falls back to the cookie", "maybe", `{"showKUNGalgameContentLimit":"nsfw"}`, false},
		{"neither", "", "", true},
	} {
		app := fiber.New()
		var gotSFW bool
		app.Get("/", func(c fiber.Ctx) error {
			gotSFW = IsSFW(c)
			return nil
		})
		req := httptest.NewRequest("GET", "/", nil)
		if tc.header != "" {
			req.Header.Set(NSFWHeader, tc.header)
		}
		if tc.cookie != "" {
			req.Header.Set("Cookie", "KUNGalgameSettings="+url.QueryEscape(tc.cookie))
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if gotSFW != tc.wantSFW {
			t.Errorf("%s: IsSFW = %v, want %v", tc.why, gotSFW, tc.wantSFW)
		}
	}
}
