package content

import "kun-galgame-api/internal/apiv1/repr"

func NewDocument(children Blocks) ContentDocument {
	return ContentDocument{Object: "document", Children: children}
}

func NewParagraph(children Inlines) ParagraphNode {
	return ParagraphNode{Object: "paragraph", Children: children}
}

func NewHeading(depth int, anchor string, children Inlines) HeadingNode {
	return HeadingNode{Object: "heading", Depth: depth, Anchor: anchor, Children: children}
}

func NewThematicBreak() ThematicBreakNode {
	return ThematicBreakNode{Object: "thematic_break"}
}

func NewBlockquote(children Blocks) BlockquoteNode {
	return BlockquoteNode{Object: "blockquote", Children: children}
}

func NewList(isOrdered bool, start *int, isSpread bool, children ListItems) ListNode {
	return ListNode{Object: "list", IsOrdered: isOrdered, Start: start, IsSpread: isSpread, Children: children}
}

func NewListItem(isChecked *bool, children Blocks) ListItemNode {
	return ListItemNode{Object: "list_item", IsChecked: isChecked, Children: children}
}

func NewCode(lang *string, value string) CodeNode {
	return CodeNode{Object: "code", Lang: lang, Value: value}
}

func NewMath(value string) MathNode {
	return MathNode{Object: "math", Value: value}
}

func NewTable(children TableRows) TableNode {
	return TableNode{Object: "table", Children: children}
}

func NewTableRow(children TableCells) TableRowNode {
	return TableRowNode{Object: "table_row", Children: children}
}

func NewTableCell(align *string, children Inlines) TableCellNode {
	return TableCellNode{Object: "table_cell", Align: align, Children: children}
}

func NewSpoiler(children Blocks) SpoilerNode {
	return SpoilerNode{Object: "spoiler", Children: children}
}

func NewText(value string) TextNode {
	return TextNode{Object: "text", Value: value}
}

func NewEmphasis(children Inlines) EmphasisNode {
	return EmphasisNode{Object: "emphasis", Children: children}
}

func NewStrong(children Inlines) StrongNode {
	return StrongNode{Object: "strong", Children: children}
}

func NewStrikethrough(children Inlines) StrikethroughNode {
	return StrikethroughNode{Object: "strikethrough", Children: children}
}

func NewInlineCode(value string) InlineCodeNode {
	return InlineCodeNode{Object: "inline_code", Value: value}
}

func NewInlineMath(value string) InlineMathNode {
	return InlineMathNode{Object: "inline_math", Value: value}
}

func NewBreak() BreakNode {
	return BreakNode{Object: "break"}
}

func NewLink(url string, children Inlines) LinkNode {
	return LinkNode{Object: "link", URL: url, Children: children}
}

func NewImage(url, alt string, image *repr.Image) ImageNode {
	return ImageNode{Object: "image", URL: url, Alt: alt, Image: image}
}

func NewVideo(url string) VideoNode {
	return VideoNode{Object: "video", URL: url}
}

func NewInlineSpoiler(children Inlines) InlineSpoilerNode {
	return InlineSpoilerNode{Object: "inline_spoiler", Children: children}
}

func NewMention(mentioned repr.UserRef) MentionNode {
	return MentionNode{Object: "mention", MentionedUser: mentioned}
}

func NewReplyReference(replyID int, floor int) ReplyReferenceNode {
	return ReplyReferenceNode{Object: "reply_reference", ReplyID: repr.ID(replyID), Floor: floor}
}
