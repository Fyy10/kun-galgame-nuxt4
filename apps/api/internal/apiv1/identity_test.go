package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

type whoamiBody struct {
	Ran    bool `json:"ran"`
	UserID int  `json:"user_id"`
}

func whoami(tier Tier, ran *atomic.Int32) func(huma.API) {
	return func(api huma.API) { registerWhoami(api, tier, ran) }
}

func registerWhoami(api huma.API, tier Tier, ran *atomic.Int32) {
	op := huma.Operation{
		OperationID: "testWhoami",
		Method:      http.MethodGet,
		Path:        "/_test/whoami",
		Summary:     "Test whoami",
		Description: "Test-only identity probe.",
	}
	switch tier {
	case TierPublic:
		op = Public(op)
	case TierOptional:
		op = Optional(op)
	case TierRequired:
		op = Required(op)
	}
	huma.Register(api, op, func(ctx context.Context, _ *struct{}) (*struct{ Body whoamiBody }, error) {
		ran.Add(1)
		out := &struct{ Body whoamiBody }{}
		out.Body.Ran = true
		if u := User(ctx); u != nil {
			out.Body.UserID = u.ID
		}
		return out, nil
	})
}

func identityFor(o middleware.IdentityOutcome) middleware.Identity {
	id := middleware.Identity{Outcome: o, Err: errors.New("underlying identity error")}
	if o == middleware.IdentitySessionOK || o == middleware.IdentityBearerOK {
		id.User = &middleware.UserInfo{ID: 7, Name: "alice"}
		id.Err = nil
	}
	return id
}

func TestPublicNeverCallsResolver(t *testing.T) {
	res := &fakeResolver{id: identityFor(middleware.IdentitySessionOK)}
	var ran atomic.Int32
	app, _ := newTestAPI(t, Deps{Resolver: res}, whoami(TierPublic, &ran))

	resp := do(t, app, http.MethodGet, "/api/v1/_test/whoami", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if res.calls != 0 {
		t.Fatalf("resolver called %d times", res.calls)
	}
	if ran.Load() != 1 {
		t.Fatalf("handler ran %d times", ran.Load())
	}
	assertCacheControl(t, resp)
}

func TestIdentityOutcomeMatrix(t *testing.T) {
	type want struct {
		status     int
		code       string
		run        bool
		retryAfter bool
		www        bool
		userID     int
	}

	cases := []struct {
		outcome middleware.IdentityOutcome
		tier    Tier
		want    want
	}{
		{middleware.IdentityAnonymous, TierOptional, want{status: 200, run: true}},
		{middleware.IdentityAnonymous, TierRequired, want{status: 401, code: problem.CodeMissingCredential, www: true}},
		{middleware.IdentitySessionMissing, TierOptional, want{status: 200, run: true}},
		{middleware.IdentitySessionMissing, TierRequired, want{status: 401, code: problem.CodeInvalidCredential, www: true}},
		{middleware.IdentitySessionStoreError, TierOptional, want{status: 503, code: problem.CodeServiceUnavailable, retryAfter: true}},
		{middleware.IdentitySessionStoreError, TierRequired, want{status: 503, code: problem.CodeServiceUnavailable, retryAfter: true}},
		{middleware.IdentitySessionRefreshDead, TierOptional, want{status: 200, run: true}},
		{middleware.IdentitySessionRefreshDead, TierRequired, want{status: 401, code: problem.CodeInvalidCredential, www: true}},
		{middleware.IdentitySessionRefreshTransient, TierOptional, want{status: 200, run: true}},
		{middleware.IdentitySessionRefreshTransient, TierRequired, want{status: 503, code: problem.CodeServiceUnavailable, retryAfter: true}},
		{middleware.IdentityBanned, TierOptional, want{status: 403, code: problem.CodeAccountBanned}},
		{middleware.IdentityBanned, TierRequired, want{status: 403, code: problem.CodeAccountBanned}},
		{middleware.IdentitySessionInternalError, TierOptional, want{status: 500, code: problem.CodeInternalError}},
		{middleware.IdentitySessionInternalError, TierRequired, want{status: 500, code: problem.CodeInternalError}},
		{middleware.IdentitySessionOK, TierOptional, want{status: 200, run: true, userID: 7}},
		{middleware.IdentitySessionOK, TierRequired, want{status: 200, run: true, userID: 7}},
		{middleware.IdentityBearerOK, TierOptional, want{status: 200, run: true, userID: 7}},
		{middleware.IdentityBearerOK, TierRequired, want{status: 200, run: true, userID: 7}},
		{middleware.IdentityBearerInvalid, TierOptional, want{status: 401, code: problem.CodeInvalidCredential, www: true}},
		{middleware.IdentityBearerInvalid, TierRequired, want{status: 401, code: problem.CodeInvalidCredential, www: true}},
		{middleware.IdentityBearerKeysUnavailable, TierOptional, want{status: 503, code: problem.CodeServiceUnavailable, retryAfter: true}},
		{middleware.IdentityBearerKeysUnavailable, TierRequired, want{status: 503, code: problem.CodeServiceUnavailable, retryAfter: true}},
		{middleware.IdentityBearerProvisioningFailed, TierOptional, want{status: 500, code: problem.CodeInternalError}},
		{middleware.IdentityBearerProvisioningFailed, TierRequired, want{status: 500, code: problem.CodeInternalError}},
	}

	for _, tc := range cases {
		name := tc.outcome.String() + "/" + string(tc.tier)
		t.Run(name, func(t *testing.T) {
			res := &fakeResolver{id: identityFor(tc.outcome)}
			var ran atomic.Int32
			app, _ := newTestAPI(t, Deps{Resolver: res}, whoami(tc.tier, &ran))

			resp := do(t, app, http.MethodGet, "/api/v1/_test/whoami", "", nil)
			if resp.StatusCode != tc.want.status {
				t.Errorf("status %d, want %d", resp.StatusCode, tc.want.status)
			}
			assertCacheControl(t, resp)
			assertRequestID(t, resp)

			if tc.want.run {
				if ran.Load() != 1 {
					t.Errorf("handler ran %d, want 1", ran.Load())
				}
				raw := readBody(t, resp)
				var body whoamiBody
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Fatalf("body %s: %v", raw, err)
				}
				if body.UserID != tc.want.userID {
					t.Errorf("user_id %d, want %d", body.UserID, tc.want.userID)
				}
			} else if ran.Load() != 0 {
				t.Errorf("handler ran %d, want 0", ran.Load())
			}

			if res.calls != 1 {
				t.Errorf("resolver calls %d, want 1", res.calls)
			}

			if tc.want.code != "" {
				p := decodeProblem(t, resp)
				if p.Code != tc.want.code {
					t.Errorf("code %s, want %s", p.Code, tc.want.code)
				}
				if strings.Contains(p.Detail, "underlying") {
					t.Errorf("detail leaked underlying error: %s", p.Detail)
				}
				if p.RequestID != resp.Header.Get(problem.HeaderRequestID) {
					t.Errorf("request_id body %s header %s", p.RequestID, resp.Header.Get(problem.HeaderRequestID))
				}
				if p.Instance == "" {
					t.Error("instance empty")
				}
			}

			gotRetry := resp.Header.Get("Retry-After") == retryAfter503
			if gotRetry != tc.want.retryAfter {
				t.Errorf("Retry-After = %q, want set=%v", resp.Header.Get("Retry-After"), tc.want.retryAfter)
			}
			gotWWW := resp.Header.Get("WWW-Authenticate") == wwwAuthenticate
			if gotWWW != tc.want.www {
				t.Errorf("WWW-Authenticate = %q, want set=%v", resp.Header.Get("WWW-Authenticate"), tc.want.www)
			}
		})
	}
}

func TestResolvedUserIsAttachedToTheFiberContext(t *testing.T) {
	res := &fakeResolver{id: identityFor(middleware.IdentitySessionOK)}
	var seen *middleware.UserInfo
	var ran atomic.Int32
	app, _ := newTestAPI(t, Deps{Resolver: res}, func(api huma.API) {
		api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
			seen = middleware.GetUser(humafiber.Unwrap(ctx))
			next(ctx)
		})
		registerWhoami(api, TierOptional, &ran)
	})

	resp := do(t, app, http.MethodGet, "/api/v1/_test/whoami", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if seen == nil || seen.ID != 7 {
		t.Fatalf("fiber locals user %+v, want id 7", seen)
	}
}
