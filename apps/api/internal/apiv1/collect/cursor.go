package collect

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"

	"kun-galgame-api/pkg/problem"
)

const cursorVersion = 1

type cursorPayload struct {
	V int      `json:"v"`
	S string   `json:"s"`
	F string   `json:"f"`
	K []string `json:"k"`
}

func Fingerprint(values ...string) string {
	h := sha256.New()
	for i, v := range values {
		if i > 0 {
			h.Write([]byte{0})
		}
		h.Write([]byte(v))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func EncodeCursor(sort, fingerprint string, keys ...string) string {
	raw, err := json.Marshal(cursorPayload{V: cursorVersion, S: sort, F: fingerprint, K: keys})
	if err != nil {
		return ""
	}
	return "cur_" + base64.RawURLEncoding.EncodeToString(raw)
}

func DecodeCursor(cur, sort, fingerprint string) ([]string, *problem.Problem) {
	if cur == "" {
		return nil, nil
	}
	const prefix = "cur_"
	if !strings.HasPrefix(cur, prefix) || len(cur) <= len(prefix) {
		return nil, invalidCursor()
	}
	raw, err := base64.RawURLEncoding.DecodeString(cur[len(prefix):])
	if err != nil || len(raw) == 0 {
		return nil, invalidCursor()
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidCursor()
	}
	if p.V != cursorVersion {
		return nil, invalidCursor()
	}
	if p.S != sort || p.F != fingerprint {
		return nil, invalidCursor()
	}
	if p.K == nil {
		p.K = []string{}
	}
	return p.K, nil
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}
