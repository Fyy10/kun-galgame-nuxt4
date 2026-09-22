package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

var reactionVocabulary = []string{
	"like", "dislike",
	"heart", "fire", "party", "love",
	"clap", "thinking", "mindblown", "scream",
	"cry", "pray", "eyes", "hundred",
	"partyface", "starstruck",
	"angry", "anxious", "banana", "eyebrow",
	"voltage", "hotdog", "hot", "sob",
	"moai", "newmoon", "police", "pouting",
	"salute", "shrimp", "halo", "sunglasses",
	"whale",
}

type ReactionInput string

func (ReactionInput) Schema(huma.Registry) *huma.Schema {
	enum := make([]any, len(reactionVocabulary))
	for i, r := range reactionVocabulary {
		enum[i] = r
	}
	n := 16
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        enum,
		MaxLength:   &n,
		Description: "Reaction token the server currently accepts. like and dislike exclude each other: setting one removes the other.",
	}
}

type TopicEngagement struct {
	Object        string            `json:"object" enum:"topic_engagement" maxLength:"16" doc:"Type discriminant. Always topic_engagement."`
	TopicID       repr.DecimalID    `json:"topic_id" doc:"Id of the topic."`
	LikeCount     int               `json:"like_count" minimum:"0" doc:"Like count. Equals the count of the like entry in reactions."`
	DislikeCount  int               `json:"dislike_count" minimum:"0" doc:"Dislike count. Equals the count of the dislike entry in reactions."`
	FavoriteCount int               `json:"favorite_count" minimum:"0" doc:"Number of users who favorited the topic."`
	UpvoteCount   int               `json:"upvote_count" minimum:"0" doc:"Number of upvotes."`
	UpvotedAt     *repr.DateTime    `json:"upvoted_at" doc:"Time of the latest upvote. null when the topic has never been upvoted."`
	Reactions     []ReactionSummary `json:"reactions" maxItems:"64" doc:"One entry per reaction token that has at least one reaction, likes and dislikes included, in first-used order. Empty array if none."`
	Viewer        TopicViewer       `json:"viewer" doc:"The caller's own state on the topic after the write."`
}

type ReplyEngagement struct {
	Object       string            `json:"object" enum:"reply_engagement" maxLength:"16" doc:"Type discriminant. Always reply_engagement."`
	ReplyID      repr.DecimalID    `json:"reply_id" doc:"Id of the reply."`
	LikeCount    int               `json:"like_count" minimum:"0" doc:"Like count. Equals the count of the like entry in reactions."`
	DislikeCount int               `json:"dislike_count" minimum:"0" doc:"Dislike count. Equals the count of the dislike entry in reactions."`
	Reactions    []ReactionSummary `json:"reactions" maxItems:"64" doc:"One entry per reaction token that has at least one reaction, likes and dislikes included, in first-used order. Empty array if none."`
	Viewer       ReplyViewer       `json:"viewer" doc:"The caller's own state on the reply after the write."`
}

type Reaction struct {
	Object    string         `json:"object" enum:"reaction" maxLength:"8" doc:"Type discriminant. Always reaction."`
	ID        repr.DecimalID `json:"id" doc:"Reaction id. JSON string of a decimal integer."`
	Reaction  ReactionToken  `json:"reaction" doc:"Reaction token, such as like, dislike, heart or clap. The vocabulary grows; show an unknown token with a neutral fallback."`
	Reactor   repr.UserRef   `json:"reactor" doc:"The user who reacted."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Time of the reaction."`
}

type TopicUpvote struct {
	Object    string         `json:"object" enum:"topic_upvote" maxLength:"12" doc:"Type discriminant. Always topic_upvote."`
	ID        repr.DecimalID `json:"id" doc:"Upvote id. JSON string of a decimal integer."`
	TopicID   repr.DecimalID `json:"topic_id" doc:"Id of the upvoted topic."`
	Upvoter   repr.UserRef   `json:"upvoter" doc:"The user who upvoted."`
	Note      *string        `json:"note" maxLength:"30" doc:"What the upvoter wrote with the upvote. null when they wrote nothing. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Time of the upvote."`
}

type UpvoteCreate struct {
	Note *string `json:"note" required:"false" maxLength:"30" doc:"A note shown with the upvote. Absent or null for none. Leading and trailing whitespace is removed, and a note of only whitespace counts as none. Free text; never use it as a decision input."`
}

type ReplyChoice struct {
	ReplyID repr.DecimalID `json:"reply_id" doc:"Id of a visible reply of this topic."`
}
