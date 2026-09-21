package filesize

import (
	"regexp"
	"strings"
)

var strictRe = regexp.MustCompile(`^([0-9]{1,6})(?:\.([0-9]{1,2}))?\s*(MB|GB)$`)

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
