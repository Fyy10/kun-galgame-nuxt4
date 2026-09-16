package service

import (
	"strings"
	"testing"
)

func TestGalCommentSnippetWindowsOnTheHit(t *testing.T) {
	body := strings.Repeat("前", 400) + "汉化补丁" + strings.Repeat("后", 400)
	got := galCommentSnippet(body, "汉化")

	if !strings.Contains(got, "汉化补丁") {
		t.Errorf("snippet dropped the hit: %q", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Errorf("snippet %q did not mark the elision", got[:12])
	}
	if n := len([]rune(got)); n > galCommentSnippetLen+1 {
		t.Errorf("snippet is %d runes, want at most %d", n, galCommentSnippetLen+1)
	}
}

func TestGalCommentSnippetStripsMarkdown(t *testing.T) {
	got := galCommentSnippet("看这张图 ![](/image/abc123) 和**汉化**说明", "汉化")
	if strings.Contains(got, "/image/") || strings.Contains(got, "**") {
		t.Errorf("snippet leaked markup: %q", got)
	}
}

// A comment shorter than the window is shown whole, with no leading ellipsis.
func TestGalCommentSnippetLeavesShortBodiesAlone(t *testing.T) {
	got := galCommentSnippet("这个汉化很棒", "汉化")
	if got != "这个汉化很棒" {
		t.Errorf("snippet = %q", got)
	}
}
