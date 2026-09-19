package content_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/infrastructure/markdown"

	"golang.org/x/net/html"
	"golang.org/x/text/width"
)

// The differential test strips all whitespace, so it passed a converter that
// turned "*word*\nnext" into "wordnext" on 18327 real bodies. This one keeps
// word boundaries and forgives only what R14 removes on purpose.
func TestConvertCorpusKeepsWordBoundaries(t *testing.T) {
	path := os.Getenv("CONTENT_CORPUS")
	if path == "" {
		t.Skip("CONTENT_CORPUS is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	conv := corpusConverter()
	var n, unexplained int
	var report strings.Builder
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var rec struct {
			Src     string `json:"src"`
			ID      int    `json:"id"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatal(err)
		}
		n++
		docs, err := conv.Convert(context.Background(), []string{rec.Content})
		if err != nil {
			t.Fatal(err)
		}
		legacy := spacing(legacySpacedText(rec.Content))
		doc := spacing(docSpacedText(docs[0]))
		if legacy == doc {
			continue
		}
		if hasNonElementTag(rec.Content) || hasUnusableImageAlt(rec.Content) {
			continue
		}
		unexplained++
		i := firstDiff(legacy, doc)
		report.WriteString(rec.Src + "/" + strconv.Itoa(rec.ID) + "\n  legacy=" + strconv.Quote(window(legacy, i, 40)) +
			"\n  doc   =" + strconv.Quote(window(doc, i, 40)) + "\n")
	}
	if err := os.WriteFile("/tmp/w1-content-go-spacing.txt", []byte(report.String()), 0o644); err != nil {
		t.Errorf("write report: %v", err)
	}
	t.Logf("corpus docs=%d spacing mismatches=%d", n, unexplained)
	if n == 0 {
		t.Fatal("corpus was empty")
	}
	if unexplained > 0 {
		t.Errorf("%d documents differ in word boundaries; see /tmp/w1-content-go-spacing.txt", unexplained)
	}
}

func TestSpacingComparisonSeesAJoinedWord(t *testing.T) {
	joined := content.NewDocument(content.Blocks{content.NewParagraph(content.Inlines{
		content.NewEmphasis(content.Inlines{content.NewText("word")}), content.NewText("next"),
	})})
	if spacing(legacySpacedText("*word*\nnext")) == spacing(docSpacedText(joined)) {
		t.Fatal("the spacing comparison accepted wordnext for word next")
	}
}

var blockTags = map[string]bool{
	"p": true, "div": true, "li": true, "ul": true, "ol": true, "tr": true, "td": true, "th": true,
	"table": true, "thead": true, "tbody": true, "blockquote": true, "pre": true, "hr": true, "br": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "video": true,
}

func legacySpacedText(src string) string {
	z := html.NewTokenizer(strings.NewReader(markdown.Render(src)))
	var b strings.Builder
	skip, mention := 0, 0
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return b.String()
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			class := ""
			for hasAttr {
				var k, v []byte
				k, v, hasAttr = z.TagAttr()
				if bytes.Equal(k, []byte("class")) {
					class = string(v)
				}
			}
			self := tt == html.SelfClosingTagToken || string(name) == "br" || string(name) == "hr" || string(name) == "img"
			if skip > 0 {
				if !self {
					skip++
				}
				continue
			}
			if strings.Contains(class, "kun-code-header") || strings.HasPrefix(class, "math ") {
				if !self {
					skip = 1
				}
				continue
			}
			if string(name) == "a" && strings.Contains(class, "kun-mention") {
				mention++
				continue
			}
			if blockTags[string(name)] {
				b.WriteByte(' ')
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			if skip > 0 {
				skip--
				continue
			}
			if mention > 0 && string(name) == "a" {
				mention--
				b.WriteString("@MENTION")
				continue
			}
			if blockTags[string(name)] {
				b.WriteByte(' ')
			}
		case html.TextToken:
			if skip == 0 && mention == 0 {
				b.Write(z.Text())
			}
		}
	}
}

func docSpacedText(doc content.ContentDocument) string {
	var b strings.Builder
	spacedBlocks(&b, doc.Children)
	return b.String()
}

func spacedBlocks(b *strings.Builder, blocks content.Blocks) {
	for _, n := range blocks {
		b.WriteByte('\n')
		switch x := n.(type) {
		case content.ParagraphNode:
			spacedInlines(b, x.Children)
		case content.HeadingNode:
			spacedInlines(b, x.Children)
		case content.BlockquoteNode:
			spacedBlocks(b, x.Children)
		case content.SpoilerNode:
			spacedBlocks(b, x.Children)
		case content.ListNode:
			for _, item := range x.Children {
				b.WriteByte('\n')
				spacedBlocks(b, item.Children)
			}
		case content.CodeNode:
			b.WriteString(x.Value)
		case content.TableNode:
			for _, row := range x.Children {
				for _, cell := range row.Children {
					b.WriteByte('\n')
					spacedInlines(b, cell.Children)
				}
			}
		}
		b.WriteByte('\n')
	}
}

func spacedInlines(b *strings.Builder, in content.Inlines) {
	for _, n := range in {
		switch x := n.(type) {
		case content.TextNode:
			b.WriteString(x.Value)
		case content.EmphasisNode:
			spacedInlines(b, x.Children)
		case content.StrongNode:
			spacedInlines(b, x.Children)
		case content.StrikethroughNode:
			spacedInlines(b, x.Children)
		case content.InlineSpoilerNode:
			spacedInlines(b, x.Children)
		case content.LinkNode:
			spacedInlines(b, x.Children)
		case content.InlineCodeNode:
			b.WriteString(x.Value)
		case content.BreakNode:
			b.WriteByte('\n')
		case content.MentionNode:
			b.WriteString("@MENTION")
		case content.ReplyReferenceNode:
			b.WriteString("#" + strconv.Itoa(x.Floor))
		}
	}
}

func spacing(s string) string {
	fields := strings.FieldsFunc(s, unicode.IsSpace)
	var b strings.Builder
	for i, f := range fields {
		if i > 0 {
			prev := []rune(fields[i-1])
			next := []rune(f)
			if !wide(prev[len(prev)-1]) && !wide(next[0]) {
				b.WriteByte(' ')
			}
		}
		b.WriteString(f)
	}
	return b.String()
}

func wide(r rune) bool {
	if unicode.Is(unicode.Hangul, r) {
		return false
	}
	k := width.LookupRune(r).Kind()
	return k == width.EastAsianWide || k == width.EastAsianFullwidth
}
