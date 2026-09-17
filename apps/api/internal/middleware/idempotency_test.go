package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

const idemKey = "0b6f7c1e-3a53-4c9e-9a4f-2f0e6c3d8a11"

type idemFixture struct {
	app     *fiber.App
	mr      *miniredis.Miniredis
	created atomic.Int32
	gate    chan struct{}
}

func newIdemFixture(t *testing.T) *idemFixture {
	t.Helper()
	f := &idemFixture{mr: miniredis.RunT(t)}
	rdb := redis.NewClient(&redis.Options{Addr: f.mr.Addr()})

	asUser := func(c fiber.Ctx) error {
		uid, _ := strconv.Atoi(c.Get("X-Test-User"))
		c.Locals(string(UserInfoKey), &UserInfo{ID: uid})
		return c.Next()
	}
	create := func(c fiber.Ctx) error {
		if f.gate != nil {
			<-f.gate
		}
		if strings.Contains(string(c.Body()), "refuse") {
			return response.Error(c, errors.ErrBadRequest("今日发帖已达上限"))
		}
		return response.OK(c, f.created.Add(1))
	}
	f.app = fiber.New()
	f.app.Post("/topic/:tid/reply", asUser, Idempotent(rdb, "reply"), create)
	return f
}

type idemResult struct {
	status   int
	replayed bool
	code     int
	data     int
}

func (f *idemFixture) post(t *testing.T, path, key, body string, uid int) idemResult {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	req.Header.Set("X-Test-User", strconv.Itoa(uid))
	if key != "" {
		req.Header.Set(IdempotencyKeyHeader, key)
	}
	resp, err := f.app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var env struct {
		Code int `json:"code"`
		Data int `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("%s: %s", path, raw)
	}
	return idemResult{
		status:   resp.StatusCode,
		replayed: resp.Header.Get(IdempotentReplayedHeader) == "true",
		code:     env.Code,
		data:     env.Data,
	}
}

func TestIdempotentReplaysTheFirstResult(t *testing.T) {
	f := newIdemFixture(t)
	first := f.post(t, "/topic/1/reply", idemKey, `{"content":"hi"}`, 7)
	again := f.post(t, "/topic/1/reply", idemKey, `{"content":"hi"}`, 7)

	if first.replayed || !again.replayed {
		t.Errorf("replayed flags = %v, %v", first.replayed, again.replayed)
	}
	if again.status != http.StatusOK || again.data != first.data || f.created.Load() != 1 {
		t.Errorf("first %+v, again %+v, created %d", first, again, f.created.Load())
	}
	if ttl := f.mr.TTL("kungal:idem:7:reply:" + idemKey); ttl != idempotencyReplayWindow {
		t.Errorf("stored result TTL = %v, want %v", ttl, idempotencyReplayWindow)
	}
}

func TestIdempotentKeyIsScopedToTheRequest(t *testing.T) {
	f := newIdemFixture(t)
	f.post(t, "/topic/1/reply", idemKey, `{"content":"hi"}`, 7)

	for _, tc := range []struct {
		why  string
		path string
		body string
	}{
		{"another body", "/topic/1/reply", `{"content":"edited"}`},
		{"another topic", "/topic/2/reply", `{"content":"hi"}`},
	} {
		got := f.post(t, tc.path, idemKey, tc.body, 7)
		if got.status != http.StatusUnprocessableEntity || got.code != errors.CodeIdempotencyMismatch {
			t.Errorf("%s: %+v", tc.why, got)
		}
	}

	if got := f.post(t, "/topic/1/reply", idemKey, `{"content":"hi"}`, 8); got.replayed {
		t.Error("another user's identical key must not replay the first user's result")
	}
	if f.created.Load() != 2 {
		t.Errorf("created %d, want 2", f.created.Load())
	}
}

func TestIdempotentReleasesARefusal(t *testing.T) {
	f := newIdemFixture(t)
	for range 2 {
		if got := f.post(t, "/topic/1/reply", idemKey, `{"content":"refuse"}`, 7); got.status != http.StatusBadRequest || got.replayed {
			t.Fatalf("%+v", got)
		}
	}
	if f.mr.Exists("kungal:idem:7:reply:" + idemKey) {
		t.Error("a refused request must not hold its key")
	}
}

func TestIdempotentInFlight(t *testing.T) {
	f := newIdemFixture(t)
	f.gate = make(chan struct{})

	var wg sync.WaitGroup
	var first idemResult
	wg.Go(func() { first = f.post(t, "/topic/1/reply", idemKey, `{"content":"hi"}`, 7) })

	key := "kungal:idem:7:reply:" + idemKey
	for !f.mr.Exists(key) {
		time.Sleep(time.Millisecond)
	}
	if ttl := f.mr.TTL(key); ttl != idempotencyPendingTTL {
		t.Errorf("pending TTL = %v, want %v", ttl, idempotencyPendingTTL)
	}
	if got := f.post(t, "/topic/1/reply", idemKey, `{"content":"hi"}`, 7); got.status != http.StatusConflict || got.code != errors.CodeIdempotencyInFlight {
		t.Errorf("concurrent retry: %+v", got)
	}

	close(f.gate)
	wg.Wait()
	if first.status != http.StatusOK || f.created.Load() != 1 {
		t.Errorf("first %+v, created %d", first, f.created.Load())
	}
}

func TestIdempotentHeaderIsOptionalButMustBeAUUID(t *testing.T) {
	f := newIdemFixture(t)
	f.post(t, "/topic/1/reply", "", `{"content":"hi"}`, 7)
	f.post(t, "/topic/1/reply", "", `{"content":"hi"}`, 7)
	if f.created.Load() != 2 {
		t.Errorf("without a key both requests must run, created %d", f.created.Load())
	}
	if got := f.post(t, "/topic/1/reply", "retry-1", `{"content":"hi"}`, 7); got.status != http.StatusBadRequest {
		t.Errorf("non-UUID key: %+v", got)
	}
}
