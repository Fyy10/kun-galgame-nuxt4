package filesize

import (
	"regexp"
	"strconv"
	"strings"
)

var strictRe = regexp.MustCompile(`^([0-9]{1,6})(?:\.([0-9]{1,2}))?\s*(MB|GB)$`)

var (
	cjkSquare = regexp.MustCompile(`【[^】]*】`)
	cjkParen  = regexp.MustCompile(`（[^）]*）`)
	paren     = regexp.MustCompile(`\([^)]*\)`)
	square    = regexp.MustCompile(`\[[^\]]*\]`)
	tokenRe   = regexp.MustCompile(`(?i)(\d{1,6}(?:[.,]\d{1,4})?)\s*\+?\s*[.\x{00B7}]?\s*(MMB|GGB|MB|GB|KB)`)
)

func Parse(text string) (string, bool) {
	text = strings.ToUpper(strings.TrimSpace(text))
	if text == "" {
		return "", false
	}
	m := strictRe.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	normalized := m[1]
	if m[2] != "" {
		frac := strings.TrimRight(m[2], "0")
		if frac != "" {
			normalized += "." + frac
		}
	}
	return normalized + " " + m[3], true
}

// Extract pulls the last "<n> MB|GB" token out of a size field that grew a
// description. The abused rows put the volume at the end ("【模拟器可玩】2GB").
func Extract(raw string) (string, bool) {
	if n, ok := Parse(raw); ok {
		return n, true
	}
	s := fold(raw)
	if n, ok := Parse(s); ok {
		return n, true
	}
	matches := tokenRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	last := matches[len(matches)-1]
	return parseToken(last[1], last[2])
}

func fold(s string) string {
	s = cjkSquare.ReplaceAllString(s, " ")
	s = cjkParen.ReplaceAllString(s, " ")
	s = paren.ReplaceAllString(s, " ")
	s = square.ReplaceAllString(s, " ")
	return s
}

func parseToken(amount, unit string) (string, bool) {
	amount = strings.ReplaceAll(amount, ",", ".")
	unit = strings.ToUpper(unit)
	switch unit {
	case "MMB":
		unit = "MB"
	case "GGB":
		unit = "GB"
	case "KB":
		return fromKB(amount)
	}
	if i := strings.IndexByte(amount, '.'); i >= 0 {
		frac := amount[i+1:]
		if len(frac) > 2 {
			amount = amount[:i+1] + frac[:2]
		}
	}
	return Parse(amount + " " + unit)
}

func fromKB(amount string) (string, bool) {
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil || f <= 0 {
		return "", false
	}
	mb := f / 1024
	if mb < 0.01 {
		mb = 0.01
	}
	s := strconv.FormatFloat(mb, 'f', 2, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return Parse(s + " MB")
}
