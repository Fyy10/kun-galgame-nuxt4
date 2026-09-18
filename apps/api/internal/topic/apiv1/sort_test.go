package apiv1

import (
	"strings"
	"testing"
)

func TestTopicStateMapsOnlyZeroAndOne(t *testing.T) {
	got, err := topicState(0)
	if err != nil || got != "published" {
		t.Fatalf("status 0: %q %v", got, err)
	}
	got, err = topicState(1)
	if err != nil || got != "hidden" {
		t.Fatalf("status 1: %q %v", got, err)
	}
	if _, err := topicState(2); err == nil {
		t.Fatal("status 2 must not map to published")
	}
	if _, err := topicState(3); err == nil {
		t.Fatal("status 3 must not map")
	}
}

func TestLikesSortTokenMapsToLikeCount(t *testing.T) {
	spec, ok := lookupSort("likes_desc")
	if !ok {
		t.Fatal("likes_desc missing")
	}
	if spec.Key != "like_count" {
		t.Fatalf("likes_desc key %q, want like_count", spec.Key)
	}
	fav, ok := lookupSort("favorites_desc")
	if !ok || fav.Key != "favorite_count" {
		t.Fatalf("favorites_desc key %+v ok=%v", fav, ok)
	}
}

func TestFingerprintIncludesNSFWAndAuth(t *testing.T) {
	base := listFingerprint("bumped_desc", "", false, false)
	if listFingerprint("bumped_desc", "", true, false) == base {
		t.Fatal("include_nsfw must affect the cursor fingerprint")
	}
	if listFingerprint("bumped_desc", "", false, true) == base {
		t.Fatal("signed-in vs anonymous must affect the cursor fingerprint")
	}
	if listFingerprint("bumped_desc", "galgame", false, false) == base {
		t.Fatal("category must affect the cursor fingerprint")
	}
	if listFingerprint("created_desc", "", false, false) == base {
		t.Fatal("sort must affect the cursor fingerprint")
	}
}

func TestSortTableHasEighteenTokens(t *testing.T) {
	if n := len(sortSpecs); n != 18 {
		t.Fatalf("sort tokens %d, want 18", n)
	}
	if _, ok := lookupSort(""); !ok {
		t.Fatal("empty token should default")
	}
	def, _ := lookupSort("")
	if def.Token != defaultSort || def.Key != "status_update_time" || def.Direction != "desc" {
		t.Fatalf("default %+v", def)
	}
	if _, ok := lookupSort("hot"); ok {
		t.Fatal("unknown token must not resolve")
	}
}

func TestSectionEnumHasTwentySeven(t *testing.T) {
	parts := strings.Split(sectionEnum, ",")
	if len(parts) != 27 {
		t.Fatalf("sections %d, want 27", len(parts))
	}
}
