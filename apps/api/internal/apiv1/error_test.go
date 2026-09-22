package apiv1

import (
	"context"
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func TestHandlerProblemCarriesRequestID(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{
			OperationID: "testFail",
			Method:      http.MethodGet,
			Path:        "/_test/fail",
			Summary:     "Test fail",
			Description: "Returns a problem.",
		}), func(context.Context, *struct{}) (*struct{}, error) {
			return nil, problem.New(problem.CodeNotFound, "no such resource")
		})
	})

	const inbound = "req_01ARZ3NDEKTSV4RRFFQ69G5FAV"
	resp := do(t, app, http.MethodGet, "/api/v1/_test/fail", "", header(problem.HeaderRequestID, inbound))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
	assertCacheControl(t, resp)
	if resp.Header.Get(problem.HeaderRequestID) != inbound {
		t.Errorf("header %s, want inbound echo", resp.Header.Get(problem.HeaderRequestID))
	}
	p := decodeProblem(t, resp)
	if p.RequestID != inbound {
		t.Errorf("body request_id %s, want %s", p.RequestID, inbound)
	}
	if p.Instance != "/api/v1/_test/fail" {
		t.Errorf("instance %s", p.Instance)
	}
	if p.Code != problem.CodeNotFound {
		t.Errorf("code %s", p.Code)
	}
}

func TestInboundRequestIDEchoedOnSuccess(t *testing.T) {
	app, _ := newTestAPI(t, Deps{})
	const inbound = "req_01ARZ3NDEKTSV4RRFFQ69G5FAV"
	resp := do(t, app, http.MethodGet, "/api/v1/problems", "", header(problem.HeaderRequestID, inbound))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	assertCacheControl(t, resp)
	if resp.Header.Get(problem.HeaderRequestID) != inbound {
		t.Errorf("header %s", resp.Header.Get(problem.HeaderRequestID))
	}
}

func TestHumaValidationBecomesProblem(t *testing.T) {
	type in struct {
		Body struct {
			Name string `json:"name" minLength:"1" maxLength:"8" doc:"Name."`
		}
	}
	app, _ := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{
			OperationID: "testValidate",
			Method:      http.MethodPost,
			Path:        "/_test/validate",
			Summary:     "Test validate",
			Description: "Validates a body.",
		}), func(context.Context, *in) (*struct{}, error) {
			return &struct{}{}, nil
		})
	})

	resp := do(t, app, http.MethodPost, "/api/v1/_test/validate", `{}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
	assertCacheControl(t, resp)
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeValidationFailed {
		t.Errorf("code %s, want VALIDATION_FAILED", p.Code)
	}
	if p.RequestID != resp.Header.Get(problem.HeaderRequestID) {
		t.Errorf("request_id mismatch body=%s header=%s", p.RequestID, resp.Header.Get(problem.HeaderRequestID))
	}
	if p.Instance == "" {
		t.Error("instance empty")
	}
}

func TestUnknownV1PathIsNotFoundProblem(t *testing.T) {
	app, _ := newTestAPI(t, Deps{})
	resp := do(t, app, http.MethodGet, "/api/v1/nope", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
	assertCacheControl(t, resp)
	id := assertRequestID(t, resp)
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeNotFound {
		t.Errorf("code %s", p.Code)
	}
	if p.RequestID != id {
		t.Errorf("request_id body %s header %s", p.RequestID, id)
	}
	if p.Instance == "" {
		t.Error("instance empty")
	}
}

func TestWrongMethodIsMethodNotAllowedProblem(t *testing.T) {
	app, _ := newTestAPI(t, Deps{})
	resp := do(t, app, http.MethodPost, "/api/v1/problems", `{}`, nil)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
	assertCacheControl(t, resp)
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeMethodNotAllowed {
		t.Errorf("code %s", p.Code)
	}
}

func TestPanickingOpIsInternalErrorProblem(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{
			OperationID: "testPanic",
			Method:      http.MethodGet,
			Path:        "/_test/panic",
			Summary:     "Test panic",
			Description: "Panics.",
		}), func(context.Context, *struct{}) (*struct{}, error) {
			panic("boom")
		})
	})

	const inbound = "req_01ARZ3NDEKTSV4RRFFQ69G5FAV"
	resp := do(t, app, http.MethodGet, "/api/v1/_test/panic", "", header(problem.HeaderRequestID, inbound))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
	assertCacheControl(t, resp)
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeInternalError {
		t.Errorf("code %s", p.Code)
	}
	if p.RequestID != inbound {
		t.Errorf("request_id %s", p.RequestID)
	}
	if p.Instance == "" {
		t.Error("instance empty")
	}
}

func TestSuccessHasCacheControl(t *testing.T) {
	app, _ := newTestAPI(t, Deps{})
	resp := do(t, app, http.MethodGet, "/api/v1/problems", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	assertCacheControl(t, resp)
	assertRequestID(t, resp)
}

func TestHandlerProblemsGetTheirStatusHeaders(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, func(api huma.API) {
		for path, code := range map[string]string{
			"/_test/unauthenticated": problem.CodeInvalidCredential,
			"/_test/unavailable":     problem.CodeServiceUnavailable,
		} {
			huma.Register(api, Public(huma.Operation{
				OperationID: "test" + code,
				Method:      http.MethodGet,
				Path:        path,
				Summary:     "Test " + code,
				Description: "Returns " + code + " from the handler.",
			}), func(context.Context, *struct{}) (*struct{}, error) {
				return nil, problem.New(code, "from the handler")
			})
		}
	})

	resp := do(t, app, http.MethodGet, "/api/v1/_test/unauthenticated", "", nil)
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != wwwAuthenticate {
		t.Errorf("401: status %d WWW-Authenticate %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}
	resp = do(t, app, http.MethodGet, "/api/v1/_test/unavailable", "", nil)
	if resp.StatusCode != http.StatusServiceUnavailable || resp.Header.Get("Retry-After") != retryAfter503 {
		t.Errorf("503: status %d Retry-After %q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
}

func TestIsV1Path(t *testing.T) {
	for path, want := range map[string]bool{
		"/api/v1":          true,
		"/api/v1/":         true,
		"/api/v1/problems": true,
		"/api/v10/topics":  false,
		"/api/v1topics":    false,
		"/api/topic/draft": false,
	} {
		if got := IsV1Path(path); got != want {
			t.Errorf("IsV1Path(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestMatchesTemplate(t *testing.T) {
	for _, tc := range []struct {
		tmpl, path string
		want       bool
	}{
		{"/topics/{topic_id}", "/topics/12", true},
		{"/topics/{topic_id}", "/topics/", false},
		{"/topics/{topic_id}", "/topics", false},
		{"/topics/{topic_id}", "/topics/12/replies", false},
		{"/problems/reasons", "/problems/reasons", true},
		{"/problems/reasons", "/problems/other", false},
	} {
		if got := matchesTemplate(tc.tmpl, tc.path); got != tc.want {
			t.Errorf("matchesTemplate(%q, %q) = %v, want %v", tc.tmpl, tc.path, got, tc.want)
		}
	}
}
