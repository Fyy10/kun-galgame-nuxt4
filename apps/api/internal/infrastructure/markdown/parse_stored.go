package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func ParseStored(source string) (ast.Node, []byte) {
	source = NormalizeStoredContent(source)
	src := []byte(source)
	reader := text.NewReader(src)
	ctx := parser.NewContext(parser.WithIDs(newUnicodeIDs()))
	root := mdContent.Parser().Parse(reader, parser.WithContext(ctx))
	return root, src
}

func HeadingID(h *ast.Heading) string {
	return headingID(h)
}
