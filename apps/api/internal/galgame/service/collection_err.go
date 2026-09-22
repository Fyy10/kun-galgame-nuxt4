package service

import (
	stderrors "errors"
	"log/slog"
	"net/http"
	"strings"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

func collectionErr(err error, fallback string) *errors.AppError {
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return errors.ErrNotFound("收藏夹不存在")
	// A grant too narrow for folder:read and an expired session are different
	// faults with different cures, and folding them together cost an outage on
	// 2026-09-08: this returned code 205, whose client-side handler re-checks
	// /api/user/status — which answers from this site's own cookie, finds it
	// perfectly healthy, and returns without a word. Every collection read
	// failed and nobody was told. Code 235 is what the five other scope-starved
	// paths here already use (playtime, cover votes, edits, submissions, image
	// upload) and its handler says the one thing that actually fixes it.
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return errors.ErrReauthRequired("收藏夹需要新的授权，请退出登录后重新登录以授予该权限")
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return errors.ErrAuthExpired()
	}
	var apiErr *catalogclient.UserAPIError
	if stderrors.As(err, &apiErr) {
		switch apiErr.Status {
		case 403:
			return errors.ErrForbidden("你没有权限操作这个收藏夹")
		case 422:
			return errors.ErrBadRequest(folderRefusalText(apiErr.Message))
		// 429 used to fall through to 500 + the caller's fallback sentence, so
		// user 90769's blown daily quota on 2026-09-20 rendered as
		// 「读取收藏夹列表失败」 — a data-corruption story for a rate limit.
		case http.StatusTooManyRequests:
			msg := "收藏夹读取过于频繁，请稍后再试"
			if strings.Contains(apiErr.Message, "Daily quota") {
				msg = "今日收藏夹读取次数已达上限，请明天再试"
			}
			return errors.New(errors.CodeBiz, msg, http.StatusTooManyRequests)
		}
	}
	slog.Error("collection: catalog call failed", "err", err)
	return errors.ErrInternal(fallback)
}

func isQuotaError(err error) bool {
	var apiErr *catalogclient.UserAPIError
	return stderrors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests
}

// Upstream answers a 422 in English, and this face has always spoken Chinese.
// A hand-maintained map over another service's prose is a bad shape in general;
// it is here because the alternative is showing readers a sentence in a
// language the rest of the page is not in, and the fallback is that sentence,
// so a phrase that changes upstream degrades rather than breaks. The two caps
// are the only 422s a request that passed this site's own validation can hit.
func folderRefusalText(upstream string) string {
	switch {
	case strings.Contains(upstream, "may keep at most"):
		return "收藏夹数量已达上限"
	case strings.Contains(upstream, "may hold at most"):
		return "这个收藏夹已经装满了"
	case strings.Contains(upstream, "default folder cannot be deleted"):
		return "默认收藏夹不能删除"
	}
	return upstream
}
