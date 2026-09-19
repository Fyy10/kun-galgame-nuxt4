package content

import (
	"encoding/json"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

const schemaRefPrefix = "#/components/schemas/"

type Block interface{ block() }

type Inline interface{ inline() }

func (ParagraphNode) block()     {}
func (HeadingNode) block()       {}
func (ThematicBreakNode) block() {}
func (BlockquoteNode) block()    {}
func (ListNode) block()          {}
func (CodeNode) block()          {}
func (MathNode) block()          {}
func (TableNode) block()         {}
func (SpoilerNode) block()       {}

func (TextNode) inline()           {}
func (EmphasisNode) inline()       {}
func (StrongNode) inline()         {}
func (StrikethroughNode) inline()  {}
func (InlineCodeNode) inline()     {}
func (InlineMathNode) inline()     {}
func (BreakNode) inline()          {}
func (LinkNode) inline()           {}
func (ImageNode) inline()          {}
func (VideoNode) inline()          {}
func (InlineSpoilerNode) inline()  {}
func (MentionNode) inline()        {}
func (ReplyReferenceNode) inline() {}

var blockMembers = []reflect.Type{
	reflect.TypeFor[ParagraphNode](),
	reflect.TypeFor[HeadingNode](),
	reflect.TypeFor[ThematicBreakNode](),
	reflect.TypeFor[BlockquoteNode](),
	reflect.TypeFor[ListNode](),
	reflect.TypeFor[CodeNode](),
	reflect.TypeFor[MathNode](),
	reflect.TypeFor[TableNode](),
	reflect.TypeFor[SpoilerNode](),
}

var inlineMembers = []reflect.Type{
	reflect.TypeFor[TextNode](),
	reflect.TypeFor[EmphasisNode](),
	reflect.TypeFor[StrongNode](),
	reflect.TypeFor[StrikethroughNode](),
	reflect.TypeFor[InlineCodeNode](),
	reflect.TypeFor[InlineMathNode](),
	reflect.TypeFor[BreakNode](),
	reflect.TypeFor[LinkNode](),
	reflect.TypeFor[ImageNode](),
	reflect.TypeFor[VideoNode](),
	reflect.TypeFor[InlineSpoilerNode](),
	reflect.TypeFor[MentionNode](),
	reflect.TypeFor[ReplyReferenceNode](),
}

type (
	Blocks     []Block
	Inlines    []Inline
	ListItems  []ListItemNode
	TableRows  []TableRowNode
	TableCells []TableCellNode
)

func (s Blocks) MarshalJSON() ([]byte, error)     { return marshalArray([]Block(s)) }
func (s Inlines) MarshalJSON() ([]byte, error)    { return marshalArray([]Inline(s)) }
func (s ListItems) MarshalJSON() ([]byte, error)  { return marshalArray([]ListItemNode(s)) }
func (s TableRows) MarshalJSON() ([]byte, error)  { return marshalArray([]TableRowNode(s)) }
func (s TableCells) MarshalJSON() ([]byte, error) { return marshalArray([]TableCellNode(s)) }

func marshalArray[T any](s []T) ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s)
}

func (Blocks) Schema(r huma.Registry) *huma.Schema {
	return &huma.Schema{Type: huma.TypeArray, Items: unionRef(r, "BlockNode",
		"A block node. Switch on object; render an unknown type's children, or its value as text.", blockMembers)}
}

func (Inlines) Schema(r huma.Registry) *huma.Schema {
	return &huma.Schema{Type: huma.TypeArray, Items: unionRef(r, "InlineNode",
		"An inline node. Switch on object; render an unknown type's children, or its value as text.", inlineMembers)}
}

func unionRef(r huma.Registry, name, doc string, members []reflect.Type) *huma.Schema {
	schemas := r.Map()
	if _, ok := schemas[name]; !ok {
		schemas[name] = &huma.Schema{}
		union := &huma.Schema{
			Description:   doc,
			Discriminator: &huma.Discriminator{PropertyName: "object", Mapping: map[string]string{}},
		}
		for _, t := range members {
			ref := r.Schema(t, true, "")
			union.OneOf = append(union.OneOf, ref)
			union.Discriminator.Mapping[Discriminant(t)] = ref.Ref
		}
		schemas[name] = union
	}
	return &huma.Schema{Ref: schemaRefPrefix + name}
}

func Discriminant(t reflect.Type) string {
	f, ok := t.FieldByName("Object")
	if !ok {
		return ""
	}
	return f.Tag.Get("enum")
}

// huma drops every component no operation references when it marshals the spec,
// and nothing returns a document until the topic detail read lands; the extension
// is the reference that keeps the node vocabulary in the committed contract.
func Register(api huma.API) {
	oapi := api.OpenAPI()
	ref := oapi.Components.Schemas.Schema(reflect.TypeFor[ContentDocument](), true, "")
	if oapi.Extensions == nil {
		oapi.Extensions = map[string]any{}
	}
	oapi.Extensions["x-content-document"] = ref
}
