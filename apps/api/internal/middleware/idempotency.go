package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	IdempotencyKeyHeader     = "Idempotency-Key"
	IdempotentReplayedHeader = "Idempotent-Replayed"
	idempotencyPrefix        = "kungal:idem:"
	idempotencyPendingTTL    = 2 * time.Minute
	idempotencyReplayWindow  = 24 * time.Hour
)

type idempotencyRecord struct {
	Fingerprint string `json:"fp"`
	Done        bool   `json:"done,omitempty"`
	Status      int    `json:"status,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Body        []byte `json:"body,omitempty"`
}

// The pending claim is short-lived so a process that dies mid-request does not
// pin the key for the whole replay window.
func Idempotent(rdb *redis.Client, scope string) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Get(IdempotencyKeyHeader)
		if raw == "" {
			return c.Next()
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return response.Error(c, errors.ErrBadRequest("Idempotency-Key 必须是 UUID"))
		}
		user, appErr := MustGetUser(c)
		if appErr != nil {
			return response.Error(c, appErr)
		}

		ctx := c.Context()
		key := fmt.Sprintf("%s%d:%s:%s", idempotencyPrefix, user.ID, scope, id)
		sum := sha256.Sum256(append([]byte(c.Method()+" "+c.Path()+"\n"), c.Body()...))
		fingerprint := hex.EncodeToString(sum[:])

		pending, _ := json.Marshal(idempotencyRecord{Fingerprint: fingerprint})
		claimed, err := rdb.SetNX(ctx, key, pending, idempotencyPendingTTL).Result()
		if err != nil {
			slog.Error("幂等键写入失败", "scope", scope, "error", err)
			return response.Error(c, errors.ErrInternal("服务器内部错误"))
		}
		if !claimed {
			return replayIdempotent(c, rdb, key, fingerprint)
		}

		if err := c.Next(); err != nil {
			rdb.Del(ctx, key)
			return err
		}
		resp := c.Response()
		if status := resp.StatusCode(); status < 200 || status >= 300 {
			rdb.Del(ctx, key)
			return nil
		}
		done, err := json.Marshal(idempotencyRecord{
			Fingerprint: fingerprint,
			Done:        true,
			Status:      resp.StatusCode(),
			ContentType: string(resp.Header.ContentType()),
			Body:        resp.Body(),
		})
		if err == nil {
			err = rdb.Set(ctx, key, done, idempotencyReplayWindow).Err()
		}
		if err != nil {
			slog.Error("幂等结果保存失败", "scope", scope, "error", err)
		}
		return nil
	}
}

func replayIdempotent(c fiber.Ctx, rdb *redis.Client, key, fingerprint string) error {
	raw, err := rdb.Get(c.Context(), key).Bytes()
	if stderrors.Is(err, redis.Nil) {
		return response.Error(c, errors.ErrIdempotencyInFlight())
	}
	var rec idempotencyRecord
	if err == nil {
		err = json.Unmarshal(raw, &rec)
	}
	if err != nil {
		slog.Error("幂等记录读取失败", "error", err)
		return response.Error(c, errors.ErrInternal("服务器内部错误"))
	}

	switch {
	case rec.Fingerprint != fingerprint:
		return response.Error(c, errors.ErrIdempotencyMismatch())
	case !rec.Done:
		return response.Error(c, errors.ErrIdempotencyInFlight())
	}
	c.Set(IdempotentReplayedHeader, "true")
	c.Set(fiber.HeaderContentType, rec.ContentType)
	return c.Status(rec.Status).Send(rec.Body)
}
