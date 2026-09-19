package markdown

import (
	"regexp"
	"testing"

	"github.com/yuin/goldmark/ast"
)

func TestParseStoredHeadingIDsMatchRender(t *testing.T) {
	cases := []string{
		"# Hello World",
		"## Hello World",
		"### Three",
		"#### Four",
		"##### Five",
		"###### Six",
		"## 引入",
		"## x\n\n## x",
		"## 引入\n\n## 引入",
		"#",
		"## ",
		"> ## Quoted",
		"- ## InList",
		"1. ## OrderedHeading",
		"# Empty-ish\n\n# Empty-ish",
		"## Hello World\n\n## Hello World\n\n## Hello World",
	}
	idRe := regexp.MustCompile(`<h[2-6] id="([^"]*)"`)
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			root, bytes := ParseStored(src)
			var got []string
			_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if !entering {
					return ast.WalkContinue, nil
				}
				h, ok := n.(*ast.Heading)
				if !ok {
					return ast.WalkContinue, nil
				}
				got = append(got, HeadingID(h))
				return ast.WalkContinue, nil
			})
			html := Render(src)
			var want []string
			for _, m := range idRe.FindAllStringSubmatch(html, -1) {
				want = append(want, m[1])
			}
			if len(got) != len(want) {
				t.Fatalf("heading count got %v from AST, want %v from Render\n html: %s\n src bytes: %q", got, want, html, bytes)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("heading %d id %q, Render id %q\n html: %s", i, got[i], want[i], html)
				}
			}
		})
	}
}

func TestParseStoredNormalizesImageTokens(t *testing.T) {
	hash, ok := LegacyStickerHash(1, 1)
	if !ok {
		t.Fatal("missing sticker 1/1")
	}
	root, src := ParseStored("![](https://sticker.kungal.com/stickers/KUNgal1/1.webp)")
	var dest string
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if img, ok := n.(*ast.Image); ok {
			dest = string(img.Destination)
		}
		return ast.WalkContinue, nil
	})
	want := "/image/" + hash + "_320"
	if dest != want {
		t.Errorf("destination %q, want %q (source %q)", dest, want, src)
	}
}
