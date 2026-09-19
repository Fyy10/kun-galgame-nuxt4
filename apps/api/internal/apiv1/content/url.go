package content

import (
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
)

const maxURLRunes = 2048

func parseUserMention(dest string) (int, bool) {
	const p = "kungal-user:"
	if !strings.HasPrefix(dest, p) {
		return 0, false
	}
	return parsePositiveInt(dest[len(p):])
}

func parseReplyRef(dest, text string) (replyID, floor int, ok bool) {
	const p = "kungal-reply:"
	if !strings.HasPrefix(dest, p) {
		return 0, 0, false
	}
	replyID, ok = parsePositiveInt(dest[len(p):])
	if !ok {
		return 0, 0, false
	}
	if strings.HasPrefix(text, "#") {
		text = text[1:]
	}
	floor, ok = parsePositiveInt(text)
	if !ok {
		return 0, 0, false
	}
	return replyID, floor, true
}

func parsePositiveInt(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func (c *converter) normalizeLinkURL(dest string) (string, bool) {
	if strings.HasPrefix(dest, "//") {
		dest = "https:" + dest
	} else if strings.HasPrefix(dest, "/") {
		dest = c.site + dest
	}
	return parseAllowedURL(dest)
}

func (c *converter) normalizeImageURL(dest string) (string, bool) {
	if strings.HasPrefix(dest, "//") {
		dest = "https:" + dest
	}
	s, ok := parseAllowedURL(dest)
	if !ok {
		return "", false
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return s, true
	default:
		return "", false
	}
}

func parseAllowedURL(dest string) (string, bool) {
	if dest == "" {
		return "", false
	}
	u, err := url.Parse(dest)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https":
		if u.Host == "" {
			return "", false
		}
	case "mailto":
	default:
		return "", false
	}
	u.Scheme = scheme
	s := u.String()
	if s == "" || utf8.RuneCountInString(s) > maxURLRunes {
		return "", false
	}
	return s, true
}

func (c *converter) imageServiceURL(dest string) (string, string, bool) {
	hash, variant, ok := markdown.ParseContentImageRef(dest)
	if !ok {
		return "", "", false
	}
	if variant != "" {
		return imageclient.VariantURL(c.cdn, hash, variant, "webp"), hash, true
	}
	return imageclient.MainURL(c.cdn, hash, "webp"), hash, true
}

func mp4VideoURL(dest string) (string, bool) {
	if strings.HasPrefix(dest, "//") {
		dest = "https:" + dest
	}
	s, ok := parseAllowedURL(dest)
	if !ok {
		return "", false
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return "", false
	}
	if !strings.HasSuffix(u.Path, ".mp4") {
		return "", false
	}
	return s, true
}
