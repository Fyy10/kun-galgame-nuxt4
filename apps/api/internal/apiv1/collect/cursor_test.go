package collect

import (
	"encoding/base64"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestCursorRoundTrip(t *testing.T) {
	fp := Fingerprint("nsfw=false", "section=g-walkthrough")
	cur := EncodeCursor("created_desc", fp, "4121", "2026-01-02T03:04:05Z")
	if !strings.HasPrefix(cur, "cur_") {
		t.Fatalf("prefix %q", cur)
	}
	keys, err := DecodeCursor(cur, "created_desc", fp)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0] != "4121" {
		t.Errorf("keys %v", keys)
	}
}

func TestCursorRejectsWrongSort(t *testing.T) {
	fp := Fingerprint("a")
	cur := EncodeCursor("a", fp, "1")
	_, p := DecodeCursor(cur, "b", fp)
	if p == nil || p.Code != problem.CodeInvalidCursor {
		t.Fatalf("code %v", p)
	}
	if len(p.Errors) != 1 || p.Errors[0].Parameter == nil || *p.Errors[0].Parameter != "cursor" {
		t.Errorf("errors %+v", p.Errors)
	}
	if p.Errors[0].Reason != problem.ReasonInvalidFormat {
		t.Errorf("reason %s", p.Errors[0].Reason)
	}
}

func TestCursorRejectsWrongFingerprint(t *testing.T) {
	cur := EncodeCursor("a", Fingerprint("x"), "1")
	_, p := DecodeCursor(cur, "a", Fingerprint("y"))
	if p == nil || p.Code != problem.CodeInvalidCursor {
		t.Fatalf("code %v", p)
	}
}

func TestCursorRejectsGarbage(t *testing.T) {
	for _, cur := range []string{"", "nope", "cur_", "cur_$$$$", "cur_not-json"} {
		if cur == "" {
			keys, p := DecodeCursor(cur, "a", "b")
			if p != nil || keys != nil {
				t.Errorf("empty cursor %v %v", keys, p)
			}
			continue
		}
		_, p := DecodeCursor(cur, "a", "b")
		if p == nil || p.Code != problem.CodeInvalidCursor {
			t.Errorf("%q → %v", cur, p)
		}
	}
}

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint("x", "y")
	b := Fingerprint("x", "y")
	c := Fingerprint("y", "x")
	if a != b || a == c || a == "" {
		t.Errorf("fp %s %s %s", a, b, c)
	}
	if Fingerprint("ab", "c") == Fingerprint("a", "bc") {
		t.Error("fingerprint does not separate its values")
	}
}

func TestCursorFromAnotherVersionIsRejected(t *testing.T) {
	cur := "cur_" + base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"s":"a","f":"b","k":["1"]}`))
	if _, p := DecodeCursor(cur, "a", "b"); p == nil || p.Code != problem.CodeInvalidCursor {
		t.Fatalf("version 2 cursor → %v", p)
	}
}
