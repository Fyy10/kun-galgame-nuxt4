package resourcevocab

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Keys []string

func (k *Keys) Scan(src any) error {
	*k = Keys{}
	switch v := src.(type) {
	case nil:
		return nil
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, (*[]string)(k))
	case string:
		if v == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), (*[]string)(k))
	default:
		return fmt.Errorf("resourcevocab: cannot scan %T into Keys", src)
	}
}

func (k Keys) Value() (driver.Value, error) {
	if k == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(k))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func Normalize(in []string, allowed map[string]int) (Keys, bool) {
	seen := map[string]bool{}
	out := Keys{}
	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if v == "others" {
			v = "other"
		}
		if _, ok := allowed[v]; !ok {
			return nil, false
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return allowed[out[i]] < allowed[out[j]] })
	return out, true
}
