package apiv1

import (
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
)

type Reply struct {
	Object            string                  `json:"object" enum:"reply" maxLength:"5" doc:"Type discriminant. Always reply."`
	ID                repr.DecimalID          `json:"id" doc:"Reply id. JSON string of a decimal integer."`
	TopicID           repr.DecimalID          `json:"topic_id" doc:"Id of the topic the reply belongs to."`
	Floor             int                     `json:"floor" minimum:"1" doc:"Floor number, assigned when the reply was created and never renumbered. Deleted and hidden replies leave gaps; a floor is not a position."`
	Author            repr.UserRef            `json:"author" doc:"Reply author."`
	AuthorMoemoepoint int                     `json:"author_moemoepoint" minimum:"-2147483648" doc:"The author's moemoepoint balance as this forum last cached it. It can be negative."`
	Content           content.ContentDocument `json:"content" doc:"Reply body as a node tree."`
	LikeCount         int                     `json:"like_count" minimum:"0" doc:"Like count. Equals the count of the like entry in reactions."`
	DislikeCount      int                     `json:"dislike_count" minimum:"0" doc:"Dislike count. Equals the count of the dislike entry in reactions."`
	Reactions         []ReactionSummary       `json:"reactions" maxItems:"64" doc:"One entry per reaction token that has at least one reaction, likes and dislikes included, in first-used order. Empty array if none."`
	IsPinned          bool                    `json:"is_pinned" doc:"Whether the topic author pinned this reply."`
	IsBestAnswer      bool                    `json:"is_best_answer" doc:"Whether this reply is marked as the topic's best answer."`
	Comments          []Comment               `json:"comments" maxItems:"1000" doc:"Comments on this reply, oldest first. Comments by banned users are left out; a comment whose parent is left out keeps its parent_comment_id. Empty array if none."`
	CreatedAt         repr.DateTime           `json:"created_at" doc:"Creation time."`
	EditedAt          *repr.DateTime          `json:"edited_at" doc:"Time of the latest edit. null when never edited."`
	Viewer            *ReplyViewer            `json:"viewer" doc:"The caller's own state on this reply. null for an anonymous caller."`
}

type ReplyViewer struct {
	HasLiked    bool `json:"has_liked" doc:"Whether the caller liked the reply."`
	HasDisliked bool `json:"has_disliked" doc:"Whether the caller disliked the reply."`
	CanEdit     bool `json:"can_edit" doc:"Whether the caller may edit the reply: its author, or staff holding the edit permission. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete   bool `json:"can_delete" doc:"Whether the caller may delete the reply: its author, or staff holding the delete permission."`
	CanLike     bool `json:"can_like" doc:"Whether the caller may like the reply: anyone but its author, while the topic is published."`
}

type Comment struct {
	Object          string                  `json:"object" enum:"comment" maxLength:"7" doc:"Type discriminant. Always comment."`
	ID              repr.DecimalID          `json:"id" doc:"Comment id. JSON string of a decimal integer."`
	ReplyID         repr.DecimalID          `json:"reply_id" doc:"Id of the reply the comment is under."`
	ParentCommentID *repr.DecimalID         `json:"parent_comment_id" doc:"Id of the comment this one answers. null for a comment on the reply itself. The parent may be absent from comments."`
	Author          repr.UserRef            `json:"author" doc:"Comment author."`
	InReplyToUser   repr.UserRef            `json:"in_reply_to_user" doc:"The user the comment answers. The server derives it when the comment is written: the parent comment's author, or the reply's author for a top-level comment. A comment written before 2026-09-22 can instead name a third party its author picked in a retired UI, so it is not always one of those two."`
	Content         content.ContentDocument `json:"content" doc:"Comment body as a node tree. A comment is plain text, so it holds at most one paragraph, and only text, break, image, mention and reply_reference nodes appear in it."`
	LikeCount       int                     `json:"like_count" minimum:"0" doc:"Like count."`
	CreatedAt       repr.DateTime           `json:"created_at" doc:"Creation time."`
	EditedAt        *repr.DateTime          `json:"edited_at" doc:"Time of the latest edit. null when never edited."`
	Viewer          *CommentViewer          `json:"viewer" doc:"The caller's own state on this comment. null for an anonymous caller."`
}

type CommentViewer struct {
	HasLiked  bool `json:"has_liked" doc:"Whether the caller liked the comment."`
	CanEdit   bool `json:"can_edit" doc:"Whether the caller may edit the comment: its author, or staff holding the edit permission, while the topic is published. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete bool `json:"can_delete" doc:"Whether the caller may delete the comment: its author, or staff holding the delete permission."`
	CanLike   bool `json:"can_like" doc:"Whether the caller may like the comment: anyone but its author."`
}

type CommentSource struct {
	Object    string         `json:"object" enum:"comment_source" maxLength:"14" doc:"Type discriminant. Always comment_source."`
	CommentID repr.DecimalID `json:"comment_id" doc:"Id of the comment."`
	ReplyID   repr.DecimalID `json:"reply_id" doc:"Id of the reply the comment is under."`
	Text      string         `json:"text" maxLength:"1007" doc:"Comment body as the stored plain text, tokens included. Free text; never use it as a decision input."`
}

type CommentCreate struct {
	Text            string          `json:"text" minLength:"1" maxLength:"1000" doc:"Comment body as plain text, stored as sent. It is never parsed as Markdown. A body of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	ParentCommentID *repr.DecimalID `json:"parent_comment_id" required:"false" doc:"Id of a visible comment under the same reply that this one answers. Absent or null for a comment on the reply itself."`
}

type CommentPatch struct {
	Text *string `json:"text,omitempty" minLength:"1" maxLength:"1000" doc:"New body as plain text. Checked as in createComment. Free text; never use it as a decision input."`
}
