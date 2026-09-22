package apiv1

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/perm"
)

type topicCaps struct {
	Edit          bool
	Hide          bool
	Unhide        bool
	Like          bool
	Upvote        bool
	SetBestAnswer bool
	PinReply      bool
}

func capsForTopic(topic *model.Topic, user *middleware.UserInfo) topicCaps {
	if topic == nil || user == nil {
		return topicCaps{}
	}
	author := user.ID == topic.UserID
	published := topic.Status == 0
	canHide := user.Can(perm.TopicHide)
	return topicCaps{
		Edit:          author || user.Can(perm.TopicEditAny),
		Hide:          published && (author || canHide),
		Unhide:        !published && ((author && topic.HiddenBy == "author") || canHide),
		Like:          published && !author,
		Upvote:        published && !author,
		SetBestAnswer: published && (author || user.Can(perm.TopicSetBestAnswer)),
		PinReply:      published && (author || user.Can(perm.ReplyPin)),
	}
}

type replyCaps struct {
	Edit   bool
	Delete bool
	Like   bool
}

func capsForReply(topic *model.Topic, reply *model.TopicReply, user *middleware.UserInfo) replyCaps {
	if topic == nil || reply == nil || user == nil {
		return replyCaps{}
	}
	author := user.ID == reply.UserID
	return replyCaps{
		Edit:   author || user.Can(perm.ReplyEditAny),
		Delete: author || user.Can(perm.ReplyDeleteAny),
		Like:   topic.Status == 0 && !author,
	}
}

type commentCaps struct {
	Edit   bool
	Delete bool
	Like   bool
}

func capsForComment(topic *model.Topic, commentAuthorID int, user *middleware.UserInfo) commentCaps {
	if topic == nil || user == nil {
		return commentCaps{}
	}
	author := user.ID == commentAuthorID
	return commentCaps{
		Edit:   topic.Status == 0 && (author || user.Can(perm.CommentTopicEdit)),
		Delete: author || user.Can(perm.CommentTopicDelete),
		Like:   !author,
	}
}
