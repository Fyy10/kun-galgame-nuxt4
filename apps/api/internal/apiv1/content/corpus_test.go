package content_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/userclient"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func TestConvertCorpusDifferential(t *testing.T) {
	path := os.Getenv("CONTENT_CORPUS")
	if path == "" {
		t.Skip("CONTENT_CORPUS is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	conv := corpusConverter()
	var n, mismatches, unexplained int
	var classes = map[string]int{}
	var buf strings.Builder
	start := time.Now()
	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec struct {
			Src     string `json:"src"`
			ID      int    `json:"id"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("corpus line: %v", err)
		}
		n++
		docs, err := conv.Convert(context.Background(), []string{rec.Content})
		if err != nil {
			t.Errorf("%s/%d convert: %v", rec.Src, rec.ID, err)
			continue
		}
		doc := docs[0]
		for _, issue := range documentIssues(doc) {
			t.Errorf("%s/%d invariant: %s", rec.Src, rec.ID, issue)
		}
		if err := schemaCheck(t, doc); err != nil {
			t.Errorf("%s/%d schema: %v", rec.Src, rec.ID, err)
		}
		htmlText := stripWS(legacyHTMLText(rec.Content))
		docText := stripWS(documentPlain(doc))
		if htmlText == docText {
			continue
		}
		mismatches++
		class := classifyMismatch(rec.Content, htmlText, docText)
		classes[class]++
		if class == "unexplained" {
			unexplained++
		}
		i := firstDiff(htmlText, docText)
		buf.WriteString(rec.Src)
		buf.WriteByte('/')
		buf.WriteString(strconv.Itoa(rec.ID))
		buf.WriteString(" class=")
		buf.WriteString(class)
		buf.WriteString(" html=")
		buf.WriteString(window(htmlText, i, 40))
		buf.WriteString(" doc=")
		buf.WriteString(window(docText, i, 40))
		buf.WriteByte('\n')
	}
	elapsed := time.Since(start)
	t.Logf("corpus docs=%d mismatches=%d unexplained=%d classes=%v elapsed=%s", n, mismatches, unexplained, classes, elapsed)
	if err := os.WriteFile("/tmp/w1-content-go-diff.txt", []byte(buf.String()), 0o644); err != nil {
		t.Errorf("write diff: %v", err)
	}
	if unexplained > 0 {
		t.Errorf("%d unexplained mismatches; see /tmp/w1-content-go-diff.txt", unexplained)
	}
	if n == 0 {
		t.Fatal("corpus was empty")
	}
}

func TestConvertCorpusPositiveControl(t *testing.T) {
	path := os.Getenv("CONTENT_CORPUS")
	if path == "" {
		t.Skip("CONTENT_CORPUS is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	conv := corpusConverter()
	var n, mismatches int
	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatal(err)
		}
		n++
		docs, err := conv.Convert(context.Background(), []string{rec.Content})
		if err != nil {
			continue
		}
		htmlText := stripWS(legacyHTMLText(rec.Content))
		docText := stripWS(documentPlainDropStrong(docs[0]))
		if htmlText != docText {
			mismatches++
		}
	}
	t.Logf("positive control mismatches=%d / %d", mismatches, n)
	if mismatches < 50 {
		t.Fatalf("positive control: dropping strong children should mismatch many documents, got %d", mismatches)
	}
}

func corpusConverter() *content.Converter {
	return &content.Converter{
		CDNBase:  testCDN,
		SiteBase: testSite,
		Images:   nil,
		Users: func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
			out := make(map[int]userclient.User, len(ids))
			for _, id := range ids {
				out[id] = userclient.User{ID: id, Name: "u" + strconv.Itoa(id)}
			}
			return out, nil
		},
	}
}

func schemaCheck(t testing.TB, doc content.ContentDocument) error {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	return documentSchema(t).Validate(v)
}

func stripWS(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func firstDiff(a, b string) int {
	ar, br := []rune(a), []rune(b)
	n := len(ar)
	if len(br) < n {
		n = len(br)
	}
	for i := 0; i < n; i++ {
		if ar[i] != br[i] {
			return i
		}
	}
	return n
}

func window(s string, i, n int) string {
	r := []rune(s)
	start := i - n
	if start < 0 {
		start = 0
	}
	end := i + n
	if end > len(r) {
		end = len(r)
	}
	return string(r[start:end])
}

func classifyMismatch(src, htmlText, docText string) string {
	if strings.Contains(docText, "kv:") && !strings.Contains(htmlText, "kv:") {
		return "R18 kv: prefix"
	}
	if hasNonElementTag(src) {
		return "R10/R11 non-element tag"
	}
	if hasUnusableImageAlt(src) {
		return "R17.3 unusable image alt"
	}
	return "unexplained"
}

func hasNonElementTag(src string) bool {
	z := html.NewTokenizer(strings.NewReader(src))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return false
		}
		if tt != html.StartTagToken && tt != html.EndTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		name, _ := z.TagName()
		if atom.Lookup(name) == 0 && len(name) > 0 {
			return true
		}
	}
}

func hasUnusableImageAlt(src string) bool {
	if strings.Contains(src, `src="data:`) || strings.Contains(src, "src=data:") {
		return true
	}
	for src != "" {
		i := strings.Index(src, "![")
		if i < 0 {
			return false
		}
		src = src[i+2:]
		j := strings.Index(src, "](")
		if j < 0 {
			return false
		}
		alt := src[:j]
		src = src[j+2:]
		k := strings.Index(src, ")")
		if k < 0 {
			return false
		}
		dest := strings.TrimSpace(src[:k])
		src = src[k+1:]
		if sp := strings.IndexAny(dest, " \t"); sp >= 0 {
			dest = dest[:sp]
		}
		if alt == "" {
			continue
		}
		if dest == "" || strings.HasPrefix(dest, "data:") {
			return true
		}
		if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") || strings.HasPrefix(dest, "//") {
			continue
		}
		if strings.HasPrefix(dest, "/image/") {
			continue
		}
		return true
	}
	return false
}

func legacyHTMLText(src string) string {
	out := markdown.Render(src)
	z := html.NewTokenizer(strings.NewReader(out))
	var b strings.Builder
	skip := 0
	mention := 0
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return b.String()
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			class := ""
			if hasAttr {
				for {
					k, v, more := z.TagAttr()
					if bytes.Equal(k, []byte("class")) {
						class = string(v)
					}
					if !more {
						break
					}
				}
			}
			self := tt == html.SelfClosingTagToken
			if skip > 0 {
				if !self {
					skip++
				}
				continue
			}
			if strings.Contains(class, "kun-code-header") {
				if !self {
					skip = 1
				}
				continue
			}
			if bytes.Equal(name, []byte("a")) && strings.Contains(class, "kun-mention") {
				mention++
				continue
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			if skip > 0 {
				skip--
				continue
			}
			if mention > 0 && bytes.Equal(name, []byte("a")) {
				mention--
				b.WriteString("@MENTION")
			}
		case html.TextToken:
			if skip > 0 || mention > 0 {
				continue
			}
			b.Write(z.Text())
		}
	}
}

func documentPlain(doc content.ContentDocument) string {
	var b strings.Builder
	writeBlocks(&b, doc.Children, false)
	return b.String()
}

func documentPlainDropStrong(doc content.ContentDocument) string {
	var b strings.Builder
	writeBlocks(&b, doc.Children, true)
	return b.String()
}

func writeBlocks(b *strings.Builder, blocks content.Blocks, dropStrong bool) {
	for _, n := range blocks {
		switch x := n.(type) {
		case content.ParagraphNode:
			writeInlines(b, x.Children, dropStrong)
		case content.HeadingNode:
			writeInlines(b, x.Children, dropStrong)
		case content.BlockquoteNode:
			writeBlocks(b, x.Children, dropStrong)
		case content.ListNode:
			for _, item := range x.Children {
				writeBlocks(b, item.Children, dropStrong)
			}
		case content.CodeNode:
			b.WriteString(x.Value)
		case content.MathNode:
			b.WriteString(`\[`)
			b.WriteString(x.Value)
			b.WriteString(`\]`)
		case content.TableNode:
			for _, row := range x.Children {
				for _, cell := range row.Children {
					writeInlines(b, cell.Children, dropStrong)
				}
			}
		case content.SpoilerNode:
			writeBlocks(b, x.Children, dropStrong)
		}
	}
}

func writeInlines(b *strings.Builder, in content.Inlines, dropStrong bool) {
	for _, n := range in {
		switch x := n.(type) {
		case content.TextNode:
			b.WriteString(x.Value)
		case content.EmphasisNode:
			writeInlines(b, x.Children, dropStrong)
		case content.StrongNode:
			if !dropStrong {
				writeInlines(b, x.Children, dropStrong)
			}
		case content.StrikethroughNode:
			writeInlines(b, x.Children, dropStrong)
		case content.InlineCodeNode:
			b.WriteString(x.Value)
		case content.InlineMathNode:
			b.WriteString(`\(`)
			b.WriteString(x.Value)
			b.WriteString(`\)`)
		case content.LinkNode:
			writeInlines(b, x.Children, dropStrong)
		case content.InlineSpoilerNode:
			writeInlines(b, x.Children, dropStrong)
		case content.MentionNode:
			b.WriteString("@MENTION")
		case content.ReplyReferenceNode:
			b.WriteByte('#')
			b.WriteString(strconv.Itoa(x.Floor))
		case content.VideoNode:
			b.WriteString("kv:")
		}
	}
}
