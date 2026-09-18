package apiv1

import (
	"testing"
	"time"

	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
)

func TestKeysetPositionRoundTrips(t *testing.T) {
	bumped, _ := lookupSort("bumped_desc")
	views, _ := lookupSort("views_desc")
	at := time.Date(2026, 1, 15, 12, 0, 0, 123456000, time.FixedZone("PST", -8*3600))
	pos, p := parseKeysetPos(encodeKeys(repository.TopicKeysetRow{ID: 7, SortTime: at}, bumped), bumped)
	if p != nil || pos.ID != 7 || !pos.SortTime.Equal(at) || !pos.TimeSort {
		t.Errorf("time key = %+v %v", pos, p)
	}
	pos, p = parseKeysetPos(encodeKeys(repository.TopicKeysetRow{ID: 9, SortInt: 42}, views), views)
	if p != nil || pos.ID != 9 || pos.SortInt != 42 || pos.TimeSort {
		t.Errorf("int key = %+v %v", pos, p)
	}
}

func TestForgedKeysetKeysAreAnInvalidCursor(t *testing.T) {
	bumped, _ := lookupSort("bumped_desc")
	views, _ := lookupSort("views_desc")
	for name, c := range map[string]struct {
		keys []string
		spec sortSpec
	}{
		"one key":         {[]string{"1"}, views},
		"three keys":      {[]string{"1", "2", "3"}, views},
		"id zero":         {[]string{"1", "0"}, views},
		"id not a number": {[]string{"1", "x"}, views},
		"int not integer": {[]string{"1.5", "2"}, views},
		"time not a time": {[]string{"yesterday", "2"}, bumped},
	} {
		if _, p := parseKeysetPos(c.keys, c.spec); p == nil || p.Code != problem.CodeInvalidCursor {
			t.Errorf("%s: %v", name, p)
		}
	}
	if pos, p := parseKeysetPos(nil, views); pos != nil || p != nil {
		t.Errorf("first page = %+v %v", pos, p)
	}
}
