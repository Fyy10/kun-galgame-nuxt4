package apiv1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"

	"github.com/alicebob/miniredis/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const testULID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

type idemIn struct {
	Status int `header:"X-Test-Status"`
	Body   struct {
		Msg string `json:"msg" maxLength:"64" doc:"Message."`
	}
}

type idemOut struct {
	Location string `header:"Location"`
	Body     struct {
		N int `json:"n"`
	}
}

type idemEnv struct {
	app     *fiber.App
	mr      *miniredis.Miniredis
	rdb     *redis.Client
	res     *fakeResolver
	created *atomic.Int32
	gate    chan struct{}
}

func newIdemEnv(t *testing.T, class IdempotencyClass) *idemEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })
	res := &fakeResolver{id: identityFor(middleware.IdentitySessionOK)}
	env := &idemEnv{mr: mr, rdb: rdb, res: res, created: &atomic.Int32{}}
	op := Required(huma.Operation{
		OperationID: "testIdem",
		Method:      http.MethodPost,
		Path:        "/_test/idem",
		Summary:     "Test idempotency",
		Description: "Idempotent test write.",
	})
	if class == idemRequired {
		op = IdempotencyRequired(op)
	} else {
		op = IdempotencyOptional(op)
	}
	env.app, _ = newTestAPI(t, Deps{Resolver: res, Redis: rdb}, func(api huma.API) {
		huma.Register(api, op, func(ctx context.Context, in *idemIn) (*idemOut, error) {
			if env.gate != nil {
				<-env.gate
			}
			switch in.Status {
			case http.StatusInternalServerError:
				return nil, problem.Internal(errNoUser)
			case http.StatusConflict:
				return nil, problem.New(problem.CodeIdempotencyKeyReused, "conflict from handler")
			case http.StatusTooManyRequests:
				p := problem.New(problem.CodeServiceUnavailable, "limited")
				p.Status = http.StatusTooManyRequests
				return nil, p
			case http.StatusUnprocessableEntity:
				return nil, problem.New(problem.CodeValidationFailed, "nope", problem.AtPointer("/msg", problem.ReasonTooShort, "too short", nil))
			}
			out := &idemOut{}
			out.Body.N = int(env.created.Add(1))
			out.Location = fmt.Sprintf("/api/v1/_test/idem/%d", out.Body.N)
			return out, nil
		})
	})
	return env
}

func (env *idemEnv) post(t *testing.T, key, body string, extra http.Header) *http.Response {
	t.Helper()
	h := header()
	if key != "" {
		h.Set(idempotencyHeader, key)
	}
	for k, vs := range extra {
		for _, v := range vs {
			h.Add(k, v)
		}
	}
	return do(t, env.app, http.MethodPost, "/api/v1/_test/idem", body, h)
}

func TestIdempotencyFirstCallAndReplay(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	key := uuid.NewString()
	first := env.post(t, key, `{"msg":"hi"}`, nil)
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first %d %s", first.StatusCode, readBody(t, first))
	}
	assertCacheControl(t, first)
	raw1 := readBody(t, first)
	if first.Header.Get(idempotencyReplayed) == "true" {
		t.Fatal("first call replayed")
	}

	again := env.post(t, key, `{"msg":"hi"}`, nil)
	if again.StatusCode != http.StatusOK {
		t.Fatalf("replay status %d", again.StatusCode)
	}
	if again.Header.Get(idempotencyReplayed) != "true" {
		t.Fatal("missing Idempotency-Replayed")
	}
	raw2 := readBody(t, again)
	if string(raw1) != string(raw2) {
		t.Errorf("replay body %s != %s", raw2, raw1)
	}
	if got, want := again.Header.Get("Location"), first.Header.Get("Location"); want == "" || got != want {
		t.Errorf("replay Location %q, want %q", got, want)
	}
	if env.created.Load() != 1 {
		t.Errorf("created %d", env.created.Load())
	}
}

func TestIdempotencyDifferentBodyConflicts(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	key := uuid.NewString()
	env.post(t, key, `{"msg":"hi"}`, nil)
	resp := env.post(t, key, `{"msg":"other"}`, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d", resp.StatusCode)
	}
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeIdempotencyKeyReused {
		t.Errorf("code %s", p.Code)
	}
}

func TestIdempotencyInFlight(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	env.gate = make(chan struct{})
	key := uuid.NewString()

	var first *http.Response
	var wg sync.WaitGroup
	wg.Go(func() {
		first = env.post(t, key, `{"msg":"hi"}`, nil)
	})
	deadline := time.Now().Add(2 * time.Second)
	for {
		found := false
		for _, k := range env.mr.Keys() {
			if len(k) > 0 {
				found = true
				break
			}
		}
		if found || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	second := env.post(t, key, `{"msg":"hi"}`, nil)
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("in-flight status %d body %s", second.StatusCode, readBody(t, second))
	}
	p := decodeProblem(t, second)
	if p.Code != problem.CodeIdempotencyRequestInProgress {
		t.Errorf("code %s", p.Code)
	}
	close(env.gate)
	wg.Wait()
	if first.StatusCode != http.StatusOK {
		t.Errorf("first %d", first.StatusCode)
	}
}

func TestIdempotencyMissingKey(t *testing.T) {
	reqd := newIdemEnv(t, idemRequired)
	resp := reqd.post(t, "", `{"msg":"hi"}`, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("required missing status %d", resp.StatusCode)
	}
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeInvalidParameter {
		t.Errorf("code %s", p.Code)
	}
	if len(p.Errors) != 1 || p.Errors[0].Header == nil || *p.Errors[0].Header != idempotencyHeader || p.Errors[0].Reason != problem.ReasonRequired {
		t.Errorf("errors %+v", p.Errors)
	}

	opt := newIdemEnv(t, idemOptional)
	ok := opt.post(t, "", `{"msg":"hi"}`, nil)
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("optional missing status %d %s", ok.StatusCode, readBody(t, ok))
	}
	ok2 := opt.post(t, "", `{"msg":"hi"}`, nil)
	if ok2.StatusCode != http.StatusOK || opt.created.Load() != 2 {
		t.Errorf("optional without key must run twice, created %d", opt.created.Load())
	}
}

func TestIdempotencyMalformedKey(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	resp := env.post(t, "not-a-key", `{"msg":"hi"}`, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeInvalidParameter {
		t.Errorf("code %s", p.Code)
	}
	if len(p.Errors) != 1 || p.Errors[0].Reason != problem.ReasonInvalidFormat {
		t.Errorf("errors %+v", p.Errors)
	}
}

func TestIdempotencyUUIDv7AndULID(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	u7, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{u7.String(), testULID} {
		resp := env.post(t, key, `{"msg":"`+key[:8]+`"}`, nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("key %s status %d %s", key, resp.StatusCode, readBody(t, resp))
			continue
		}
		_ = readBody(t, resp)
	}
}

func TestIdempotencyReleases5xx409429(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	key := uuid.NewString()
	cases := []struct {
		status int
		header http.Header
	}{
		{500, header("X-Test-Status", "500")},
		{409, header("X-Test-Status", "409")},
		{429, header("X-Test-Status", "429")},
	}
	for _, tc := range cases {
		resp := env.post(t, key, `{"msg":"x"}`, tc.header)
		if resp.StatusCode != tc.status {
			t.Errorf("status %d, want %d body %s", resp.StatusCode, tc.status, readBody(t, resp))
		}
		ok := env.post(t, key, `{"msg":"x"}`, nil)
		if ok.StatusCode != http.StatusOK {
			t.Errorf("after %d, retry status %d %s", tc.status, ok.StatusCode, readBody(t, ok))
		}
		key = uuid.NewString()
	}
}

func TestIdempotencyStores4xxAndReplays(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	key := uuid.NewString()
	first := env.post(t, key, `{"msg":"x"}`, header("X-Test-Status", "422"))
	if first.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("first %d %s", first.StatusCode, readBody(t, first))
	}
	raw1 := readBody(t, first)
	again := env.post(t, key, `{"msg":"x"}`, header("X-Test-Status", "422"))
	if again.Header.Get(idempotencyReplayed) != "true" {
		t.Fatal("expected replay")
	}
	if again.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("replay status %d", again.StatusCode)
	}
	if string(readBody(t, again)) != string(raw1) {
		t.Error("replay body mismatch")
	}
}

func TestIdempotencyUsersDoNotCollide(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	key := uuid.NewString()
	first := env.post(t, key, `{"msg":"hi"}`, nil)
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first %d", first.StatusCode)
	}
	env.res.id.User = &middleware.UserInfo{ID: 8, Name: "bob"}
	second := env.post(t, key, `{"msg":"hi"}`, nil)
	if second.Header.Get(idempotencyReplayed) == "true" {
		t.Fatal("other user replayed the first user's result")
	}
	if env.created.Load() != 2 {
		t.Errorf("created %d, want 2", env.created.Load())
	}
}

func TestIdempotencyIgnoresLegacyPrefix(t *testing.T) {
	env := newIdemEnv(t, idemRequired)
	key := uuid.NewString()
	legacy, err := json.Marshal(idempotencyRecord{
		Fingerprint: "deadbeef",
		Done:        true,
		Status:      200,
		ContentType: "application/json",
		Body:        []byte(`{"n":99}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := env.rdb.Set(t.Context(), "kungal:idem:7:testIdem:"+key, legacy, time.Hour).Err(); err != nil {
		t.Fatal(err)
	}
	resp := env.post(t, key, `{"msg":"hi"}`, nil)
	if resp.Header.Get(idempotencyReplayed) == "true" {
		t.Fatal("read a legacy kungal:idem: record")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d %s", resp.StatusCode, readBody(t, resp))
	}
	var body struct {
		N int `json:"n"`
	}
	if err := json.Unmarshal(readBody(t, resp), &body); err != nil {
		t.Fatal(err)
	}
	if body.N != 1 {
		t.Errorf("n %d, want 1", body.N)
	}
}

func TestIdempotencyHelperRefusesNonRequired(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = IdempotencyRequired(Optional(huma.Operation{
		OperationID: "nope",
		Method:      http.MethodPost,
		Path:        "/nope",
	}))
}
