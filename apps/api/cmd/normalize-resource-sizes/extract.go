package main

import (
	"regexp"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/filesize"
)

var (
	cjkSquare = regexp.MustCompile(`【[^】]*】`)
	cjkParen  = regexp.MustCompile(`（[^）]*）`)
	paren     = regexp.MustCompile(`\([^)]*\)`)
	square    = regexp.MustCompile(`\[[^\]]*\]`)
	tokenRe   = regexp.MustCompile(`(?i)(\d{1,6}(?:[.,]\d{1,4})?)\s*\+?\s*[.\x{00B7}]?\s*(MMB|GGB|MB|GB|KB)`)
)

func extract(raw string) (string, bool) {
	if n, ok := filesize.Parse(raw); ok {
		return n, true
	}
	s := fold(raw)
	if n, ok := filesize.Parse(s); ok {
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
	return filesize.Parse(amount + " " + unit)
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
	return filesize.Parse(s + " MB")
}
