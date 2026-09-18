package apiv1

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type fakeResolver struct {
	calls int
	id    middleware.Identity
	hook  func()
}

func (f *fakeResolver) ResolveIdentity(fiber.Ctx) middleware.Identity {
	f.calls++
	if f.hook != nil {
		f.hook()
	}
	return f.id
}

func newTestAPI(t *testing.T, deps Deps, registrars ...func(huma.API)) (*fiber.App, huma.API) {
	t.Helper()
	app := fiber.New(fiber.Config{ErrorHandler: WriteFiberError})
	app.Use(recover.New())
	api := Setup(app, deps, registrars...)
	return app, api
}

func do(t *testing.T, app *fiber.App, method, path, body string, hdr http.Header) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 8 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type wireProblem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
	Errors    []struct {
		Header    *string `json:"header"`
		Parameter *string `json:"parameter"`
		Pointer   *string `json:"pointer"`
		Reason    string  `json:"reason"`
		Detail    string  `json:"detail"`
	} `json:"errors"`
}

func decodeProblem(t *testing.T, resp *http.Response) wireProblem {
	t.Helper()
	raw := readBody(t, resp)
	var p wireProblem
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("decode problem: %v\n%s", err, raw)
	}
	return p
}

func assertCacheControl(t *testing.T, resp *http.Response) {
	t.Helper()
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func assertRequestID(t *testing.T, resp *http.Response) string {
	t.Helper()
	id := resp.Header.Get(problem.HeaderRequestID)
	if !problem.ValidRequestID(id) {
		t.Errorf("X-Request-ID = %q, want req_ + ULID", id)
	}
	return id
}

func header(kv ...string) http.Header {
	h := make(http.Header)
	for i := 0; i+1 < len(kv); i += 2 {
		h.Add(kv[i], kv[i+1])
	}
	return h
}
