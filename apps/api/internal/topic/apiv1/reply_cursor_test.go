package apiv1

import (
	"testing"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/pkg/problem"
)

func TestParseReplyPos(t *testing.T) {
	pos, p := parseReplyPos(nil)
	if pos != nil || p != nil {
		t.Fatalf("first page = %+v %v", pos, p)
	}
	pos, p = parseReplyPos([]string{"3", "9"})
	if p != nil || pos == nil || pos.Floor != 3 || pos.ID != 9 {
		t.Fatalf("ok = %+v %v", pos, p)
	}
	for name, keys := range map[string][]string{
		"one":   {"1"},
		"three": {"1", "2", "3"},
		"id 0":  {"1", "0"},
		"bad":   {"x", "2"},
		"floor": {"1.5", "2"},
	} {
		if _, p := parseReplyPos(keys); p == nil || p.Code != problem.CodeInvalidCursor {
			t.Errorf("%s: %v", name, p)
		}
	}
}

func TestReplyCursorFingerprintIncludesTopicAndFromFloor(t *testing.T) {
	base := collect.Fingerprint("1", "0")
	if collect.Fingerprint("2", "0") == base {
		t.Fatal("topic id must affect the fingerprint")
	}
	if collect.Fingerprint("1", "4") == base {
		t.Fatal("from_floor must affect the fingerprint")
	}
}

func TestReplySortDir(t *testing.T) {
	dir, ok := replySortDir("")
	if !ok || dir != "asc" {
		t.Fatalf("default %q %v", dir, ok)
	}
	dir, ok = replySortDir("floor_desc")
	if !ok || dir != "desc" {
		t.Fatalf("desc %q %v", dir, ok)
	}
	if _, ok := replySortDir("nope"); ok {
		t.Fatal("unknown")
	}
}
