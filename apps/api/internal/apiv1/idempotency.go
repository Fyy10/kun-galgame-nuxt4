package apiv1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type IdempotencyClass string

const (
	idemNone     IdempotencyClass = ""
	idemOptional IdempotencyClass = "optional"
	idemRequired IdempotencyClass = "required"
)

const (
	metaIdempotency         = "apiv1.idempotency"
	idempotencyHeader       = "Idempotency-Key"
	idempotencyReplayed     = "Idempotency-Replayed"
	idempotencyPrefix       = "kungal:idem:v1:"
	idempotencyPendingTTL   = 2 * time.Minute
	idempotencyReplayWindow = 24 * time.Hour
	idempotencyKeyPattern   = `^([0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}|[0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{26})$`
	idempotencyKeyDoc       = "Caller-generated UUID (canonical 8-4-4-4-12 hex, any version) or 26-character Crockford ULID. Scoped to (user, operation, key) for 24 hours."
)

var (
	uuidIdempotencyKey = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	ulidIdempotencyKey = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{26}$`)
)

type idempotencyRecord struct {
	Fingerprint string `json:"fp"`
	Done        bool   `json:"done,omitempty"`
	Status      int    `json:"status,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Location    string `json:"location,omitempty"`
	Body        []byte `json:"body,omitempty"`
}

func IdempotencyOptional(op huma.Operation) huma.Operation {
	return withIdempotency(op, idemOptional)
}

func IdempotencyRequired(op huma.Operation) huma.Operation {
	return withIdempotency(op, idemRequired)
}

func withIdempotency(op huma.Operation, class IdempotencyClass) huma.Operation {
	if TierFromOp(&op) != TierRequired {
		panic("apiv1: idempotency is only legal on required-tier operations")
	}
	if op.Metadata == nil {
		op.Metadata = map[string]any{}
	}
	op.Metadata[metaIdempotency] = class
	minLen, maxLen := 26, 36
	op.Parameters = append(op.Parameters, &huma.Param{
		Name:        idempotencyHeader,
		In:          "header",
		Required:    class == idemRequired,
		Description: idempotencyKeyDoc,
		Schema: &huma.Schema{
			Type:        "string",
			MinLength:   &minLen,
			MaxLength:   &maxLen,
			Pattern:     idempotencyKeyPattern,
			Description: idempotencyKeyDoc,
		},
	})
	addErrors(&op, http.StatusBadRequest, http.StatusConflict)
	return op
}

func validIdempotencyKey(s string) bool {
	return uuidIdempotencyKey.MatchString(s) || ulidIdempotencyKey.MatchString(s)
}

func newIdempotencyMiddleware(rdb *redis.Client) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		op := ctx.Operation()
		class := idempotencyClassOf(op)
		if class == idemNone {
			next(ctx)
			return
		}

		fc := humafiber.Unwrap(ctx)
		raw := fc.Get(idempotencyHeader)
		if raw == "" {
			if class == idemRequired {
				writeProblem(ctx, problem.New(
					problem.CodeInvalidParameter,
					"Idempotency-Key is required.",
					problem.AtHeader(idempotencyHeader, problem.ReasonRequired, "Idempotency-Key is required.", nil),
				))
				return
			}
			next(ctx)
			return
		}
		if !validIdempotencyKey(raw) {
			writeProblem(ctx, problem.New(
				problem.CodeInvalidParameter,
				"Idempotency-Key must be a UUID or ULID.",
				problem.AtHeader(idempotencyHeader, problem.ReasonInvalidFormat, "Idempotency-Key must be a canonical UUID or a 26-character Crockford ULID.", nil),
			))
			return
		}

		user := User(ctx.Context())
		if user == nil {
			writeProblem(ctx, problem.Internal(errNoUser))
			return
		}
		if rdb == nil {
			writeProblem(ctx, problem.New(problem.CodeServiceUnavailable, "A dependency is unavailable. Retry the request."))
			return
		}

		opID := ""
		if op != nil {
			opID = op.OperationID
		}
		key := fmt.Sprintf("%s%d:%s:%s", idempotencyPrefix, user.ID, opID, raw)
		sum := sha256.Sum256(append([]byte(fc.Method()+" "+fc.Path()+"\n"), fc.Body()...))
		fingerprint := hex.EncodeToString(sum[:])
		rctx := fc.Context()

		pending, err := json.Marshal(idempotencyRecord{Fingerprint: fingerprint})
		if err != nil {
			writeProblem(ctx, problem.Internal(err))
			return
		}
		claimed, err := rdb.SetNX(rctx, key, pending, idempotencyPendingTTL).Result()
		if err != nil {
			slog.Error("apiv1 idempotency claim", "request_id", problem.RequestID(fc), "err", err)
			writeProblem(ctx, problem.New(problem.CodeServiceUnavailable, "A dependency is unavailable. Retry the request."))
			return
		}
		if !claimed {
			replayIdempotent(ctx, fc, rdb, key, fingerprint)
			return
		}

		stored := false
		defer func() {
			if !stored {
				if err := rdb.Del(rctx, key).Err(); err != nil {
					slog.Error("apiv1 idempotency release", "request_id", problem.RequestID(fc), "err", err)
				}
			}
		}()

		next(ctx)

		resp := fc.Response()
		status := resp.StatusCode()
		if !storeIdempotentStatus(status) {
			return
		}
		done, err := json.Marshal(idempotencyRecord{
			Fingerprint: fingerprint,
			Done:        true,
			Status:      status,
			ContentType: string(resp.Header.ContentType()),
			Location:    string(resp.Header.Peek(fiber.HeaderLocation)),
			Body:        append([]byte(nil), resp.Body()...),
		})
		if err != nil {
			slog.Error("apiv1 idempotency encode", "request_id", problem.RequestID(fc), "err", err)
			return
		}
		if err := rdb.Set(rctx, key, done, idempotencyReplayWindow).Err(); err != nil {
			slog.Error("apiv1 idempotency store", "request_id", problem.RequestID(fc), "err", err)
			return
		}
		stored = true
	}
}

func storeIdempotentStatus(status int) bool {
	if status < 200 || status > 499 {
		return false
	}
	if status == http.StatusConflict || status == http.StatusTooManyRequests {
		return false
	}
	return true
}

func replayIdempotent(ctx huma.Context, fc fiber.Ctx, rdb *redis.Client, key, fingerprint string) {
	raw, err := rdb.Get(fc.Context(), key).Bytes()
	if errors.Is(err, redis.Nil) {
		writeProblem(ctx, problem.New(problem.CodeIdempotencyRequestInProgress, "A request with the same Idempotency-Key is still being processed."))
		return
	}
	if err != nil {
		slog.Error("apiv1 idempotency read", "request_id", problem.RequestID(fc), "err", err)
		writeProblem(ctx, problem.New(problem.CodeServiceUnavailable, "A dependency is unavailable. Retry the request."))
		return
	}
	var rec idempotencyRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		writeProblem(ctx, problem.Internal(err))
		return
	}
	switch {
	case rec.Fingerprint != fingerprint:
		writeProblem(ctx, problem.New(problem.CodeIdempotencyKeyReused, "The same Idempotency-Key was sent with a different request."))
	case !rec.Done:
		writeProblem(ctx, problem.New(problem.CodeIdempotencyRequestInProgress, "A request with the same Idempotency-Key is still being processed."))
	default:
		fc.Set(idempotencyReplayed, "true")
		if rec.ContentType != "" {
			fc.Set(fiber.HeaderContentType, rec.ContentType)
		}
		if rec.Location != "" {
			fc.Set(fiber.HeaderLocation, rec.Location)
		}
		if err := fc.Status(rec.Status).Send(rec.Body); err != nil {
			slog.Error("apiv1 idempotency replay", "request_id", problem.RequestID(fc), "err", err)
		}
	}
}

func idempotencyClassOf(op *huma.Operation) IdempotencyClass {
	if op == nil || op.Metadata == nil {
		return idemNone
	}
	c, _ := op.Metadata[metaIdempotency].(IdempotencyClass)
	return c
}

var errNoUser = errors.New("apiv1: idempotency requires an authenticated user")
