package content

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
	"golang.org/x/text/width"
)

const (
	maxValueRunes = 100007
	maxAltRunes   = 512
	maxAnchor     = 128
)

func decodeText(source []byte) string {
	if len(source) == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(source))
	writeDecoded(&b, source)
	return b.String()
}

func writeDecoded(b *strings.Builder, source []byte) {
	escaped := false
	var ok bool
	limit := len(source)
	n := 0
	for i := 0; i < limit; i++ {
		c := source[i]
		if escaped {
			if util.IsPunct(c) {
				b.Write(source[n : i-1])
				n = i
				escaped = false
				continue
			}
		}
		if c == '\x00' {
			b.Write(source[n:i])
			b.WriteRune('\uFFFD')
			n = i + 1
			escaped = false
			continue
		}
		if c == '&' {
			pos := i
			next := i + 1
			if next < limit && source[next] == '#' {
				nnext := next + 1
				if nnext < limit {
					nc := source[nnext]
					if nnext < limit && nc == 'x' || nc == 'X' {
						start := nnext + 1
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsHexDecimal)
						if ok && i < limit && source[i] == ';' && i-start < 7 {
							v, _ := strconv.ParseUint(util.BytesToReadOnlyString(source[start:i]), 16, 32)
							b.Write(source[n:pos])
							n = i + 1
							b.WriteRune(util.ToValidRune(rune(v)))
							continue
						}
					} else if nc >= '0' && nc <= '9' {
						start := nnext
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsNumeric)
						if ok && i < limit && i-start < 8 && source[i] == ';' {
							v, _ := strconv.ParseUint(util.BytesToReadOnlyString(source[start:i]), 10, 32)
							b.Write(source[n:pos])
							n = i + 1
							b.WriteRune(util.ToValidRune(rune(v)))
							continue
						}
					}
				}
			} else {
				start := next
				i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsAlphaNumeric)
				if ok && i < limit && source[i] == ';' {
					name := util.BytesToReadOnlyString(source[start:i])
					entity, found := util.LookUpHTML5EntityByName(name)
					if found {
						b.Write(source[n:pos])
						n = i + 1
						b.Write(entity.Characters)
						continue
					}
				}
			}
			i = next - 1
		}
		if c == '\\' {
			escaped = true
			continue
		}
		escaped = false
	}
	b.Write(source[n:])
}

func (c *converter) textValue(n *ast.Text) string {
	raw := n.Value(c.src)
	if n.IsRaw() {
		return string(raw)
	}
	return decodeText(raw)
}

func (c *converter) stringValue(n *ast.String) string {
	if n.IsRaw() || n.IsCode() {
		return string(n.Value)
	}
	return decodeText(n.Value)
}

func codeSpanValue(n *ast.CodeSpan, source []byte) string {
	var b strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		t, ok := child.(*ast.Text)
		if !ok {
			continue
		}
		value := t.Segment.Value(source)
		if len(value) > 0 && value[len(value)-1] == '\n' {
			b.Write(value[:len(value)-1])
			b.WriteByte(' ')
		} else {
			b.Write(value)
		}
	}
	return truncateRunes(b.String(), maxValueRunes)
}

func nodeLines(n ast.Node, source []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(source))
	}
	s := b.String()
	s = strings.TrimSuffix(s, "\n")
	return s
}

func fenceLang(info string) *string {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return nil
	}
	lang := strings.ToLower(fields[0])
	if lang == "" || utf8.RuneCountInString(lang) > 32 {
		return nil
	}
	for _, r := range lang {
		if unicode.IsSpace(r) {
			return nil
		}
	}
	return &lang
}

func truncateRunes(s string, n int) string {
	if n <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}

func (c *converter) plainText(n ast.Node) string {
	var b strings.Builder
	_ = ast.Walk(n, func(x ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := x.(type) {
		case *ast.Text:
			b.WriteString(c.textValue(t))
		case *ast.String:
			b.WriteString(c.stringValue(t))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func mergeInlines(in Inlines) Inlines {
	if len(in) == 0 {
		return in
	}
	out := make(Inlines, 0, len(in))
	for _, n := range in {
		t, ok := n.(TextNode)
		if !ok {
			out = append(out, n)
			continue
		}
		if t.Value == "" {
			continue
		}
		if len(out) > 0 {
			if prev, ok := out[len(out)-1].(TextNode); ok {
				out[len(out)-1] = NewText(prev.Value + t.Value)
				continue
			}
		}
		out = append(out, t)
	}
	for i, n := range out {
		if t, ok := n.(TextNode); ok {
			out[i] = NewText(truncateRunes(t.Value, maxValueRunes))
		}
	}
	return out
}

func eastAsianWide(r rune) bool {
	k := width.LookupRune(r).Kind()
	return k == width.EastAsianWide || k == width.EastAsianFullwidth
}

func removeSoftBreak(left, right rune) bool {
	if unicode.Is(unicode.Hangul, left) || unicode.Is(unicode.Hangul, right) {
		return false
	}
	return eastAsianWide(left) && eastAsianWide(right)
}

func lastRune(s string) (rune, bool) {
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError {
		return 0, false
	}
	return r, true
}

func firstRune(s string) (rune, bool) {
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return 0, false
	}
	return r, true
}

type neighbor int

const (
	neighborNone neighbor = iota
	neighborRune
	neighborOther
	neighborLineBreak
)

// A soft break is often the last thing inside an inline container: in
// "*word*\nnext" it sits on the text inside the emphasis, so looking only at
// siblings found nothing and joined the two words into "wordnext".
func (c *converter) runeAfter(n ast.Node) (rune, neighbor) {
	for cur := n; cur != nil; cur = cur.Parent() {
		for s := cur.NextSibling(); s != nil; s = s.NextSibling() {
			if r, k := c.edgeRune(s, true); k != neighborNone {
				return r, k
			}
		}
		if p := cur.Parent(); p == nil || p.Type() != ast.TypeInline {
			break
		}
	}
	return 0, neighborNone
}

func (c *converter) runeBefore(n ast.Node) (rune, neighbor) {
	for cur := n; cur != nil; cur = cur.Parent() {
		for s := cur.PreviousSibling(); s != nil; s = s.PreviousSibling() {
			if r, k := c.edgeRune(s, false); k != neighborNone {
				return r, k
			}
		}
		if p := cur.Parent(); p == nil || p.Type() != ast.TypeInline {
			break
		}
	}
	return 0, neighborNone
}

func (c *converter) edgeRune(n ast.Node, first bool) (rune, neighbor) {
	pick := func(s string) (rune, neighbor) {
		var r rune
		var ok bool
		if first {
			r, ok = firstRune(s)
		} else {
			r, ok = lastRune(s)
		}
		if !ok {
			return 0, neighborNone
		}
		return r, neighborRune
	}
	switch t := n.(type) {
	case *ast.Text:
		return pick(c.textValue(t))
	case *ast.String:
		return pick(c.stringValue(t))
	case *ast.CodeSpan:
		return pick(codeSpanValue(t, c.src))
	case *ast.AutoLink:
		return pick(string(t.Label(c.src)))
	case *ast.RawHTML:
		if brTagRe.Match(t.Segments.Value(c.src)) {
			return 0, neighborLineBreak
		}
		return 0, neighborOther
	case *ast.Image:
		return 0, neighborOther
	}
	if n.FirstChild() == nil {
		if n.Type() == ast.TypeInline {
			return 0, neighborOther
		}
		return 0, neighborNone
	}
	if first {
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if r, k := c.edgeRune(child, true); k != neighborNone {
				return r, k
			}
		}
	} else {
		for child := n.LastChild(); child != nil; child = child.PreviousSibling() {
			if r, k := c.edgeRune(child, false); k != neighborNone {
				return r, k
			}
		}
	}
	return 0, neighborNone
}
