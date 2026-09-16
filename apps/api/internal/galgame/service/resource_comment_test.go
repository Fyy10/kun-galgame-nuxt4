package service

import (
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func TestResourceAnchorID(t *testing.T) {
	cases := []struct {
		src  CommentSource
		id   int
		want string
	}{
		{sourceRating, 42, "rating:42"},
		{sourceWebsite, 7, "website:7"},
		{sourceToolset, 3, "toolset:3"},
	}
	for _, tc := range cases {
		if got := tc.src.anchorID(tc.id); got != tc.want {
			t.Errorf("%s.anchorID(%d) = %q, want %q", tc.src.key, tc.id, got, tc.want)
		}
	}
}

func TestResourceSourceMetadata(t *testing.T) {
	cases := []struct {
		src      CommentSource
		key      string
		feedType string
	}{
		{SourceRating(), "rating", "GALGAME_RATING_COMMENT_CREATION"},
		{SourceWebsite(), "website", "GALGAME_WEBSITE_COMMENT_CREATION"},
		{SourceToolset(), "toolset", "TOOLSET_COMMENT_CREATION"},
	}
	for _, tc := range cases {
		if tc.src.key != tc.key || tc.src.feedType != tc.feedType {
			t.Errorf("source %+v, want key=%q feedType=%q", tc.src, tc.key, tc.feedType)
		}
	}
}

func TestShouldNotifyOwner(t *testing.T) {
	cases := []struct {
		name         string
		src          CommentSource
		sender       int
		owner        int
		isReply      bool
		hasPref      bool
		lookupFailed bool
		want         bool
	}{
		{"toolset top-level no pref", sourceToolset, 1, 9, false, false, false, true},
		{"toolset reply", sourceToolset, 1, 9, true, false, false, false},
		{"toolset owner is author", sourceToolset, 9, 9, false, false, false, false},
		{"toolset pref present", sourceToolset, 1, 9, false, true, false, false},
		{"toolset lookup error", sourceToolset, 1, 9, false, false, true, true},
		{"toolset lookup error still not reply", sourceToolset, 1, 9, true, false, true, false},
		{"resource top-level no pref", sourceResource, 1, 9, false, false, false, true},
		{"quiz top-level no pref", sourceQuiz, 1, 9, false, false, false, true},
		{"rating never", sourceRating, 1, 9, false, false, false, false},
		{"website never", sourceWebsite, 1, 9, false, false, false, false},
		{"no owner", sourceToolset, 1, 0, false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldNotifyOwner(tc.src, tc.sender, tc.owner, tc.isReply, tc.hasPref, tc.lookupFailed)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestContainsPost(t *testing.T) {
	posts := []communityclient.PostView{{ID: 10}, {ID: 20}, {ID: 30}}
	if !containsPost(posts, 20) {
		t.Error("containsPost should find 20")
	}
	if containsPost(posts, 99) {
		t.Error("containsPost should not find 99")
	}
	if containsPost(nil, 1) {
		t.Error("containsPost(nil) should be false")
	}
}
