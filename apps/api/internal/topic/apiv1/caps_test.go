package apiv1

import (
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
)

func TestCapsForTopic(t *testing.T) {
	const author, other, staff = 1, 2, 3
	published := &model.Topic{ID: 10, UserID: author}
	hiddenByAuthor := &model.Topic{ID: 11, UserID: author, Status: 1, HiddenBy: "author"}
	hiddenByModerator := &model.Topic{ID: 12, UserID: author, Status: 1, HiddenBy: "moderator"}
	hiddenByTrust := &model.Topic{ID: 13, UserID: author, Status: 1, HiddenBy: "trust"}
	authorUser := &middleware.UserInfo{ID: author, Roles: []string{"user"}}
	otherUser := &middleware.UserInfo{ID: other, Roles: []string{"user"}}
	staffUser := &middleware.UserInfo{ID: staff, Roles: []string{"moderator"}}

	cases := []struct {
		name  string
		topic *model.Topic
		user  *middleware.UserInfo
		want  topicCaps
	}{
		{"anonymous", published, nil, topicCaps{}},
		{"author on published", published, authorUser, topicCaps{Edit: true, Hide: true, SetBestAnswer: true, PinReply: true}},
		{"other on published", published, otherUser, topicCaps{Like: true, Upvote: true}},
		{"staff on published", published, staffUser, topicCaps{Edit: true, Hide: true, Like: true, Upvote: true, SetBestAnswer: true, PinReply: true}},
		{"author on own hide", hiddenByAuthor, authorUser, topicCaps{Edit: true, Unhide: true}},
		{"author on moderator hide", hiddenByModerator, authorUser, topicCaps{Edit: true}},
		{"author on trust hide", hiddenByTrust, authorUser, topicCaps{Edit: true}},
		{"staff on moderator hide", hiddenByModerator, staffUser, topicCaps{Edit: true, Unhide: true}},
		{"other on hidden", hiddenByAuthor, otherUser, topicCaps{}},
	}
	for _, c := range cases {
		if got := capsForTopic(c.topic, c.user); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestCapsForReply(t *testing.T) {
	const author, other, staff = 1, 2, 3
	published := &model.Topic{ID: 10, UserID: other}
	hidden := &model.Topic{ID: 11, UserID: other, Status: 1, HiddenBy: "author"}
	reply := &model.TopicReply{ID: 20, TopicID: 10, UserID: author}
	authorUser := &middleware.UserInfo{ID: author, Roles: []string{"user"}}
	otherUser := &middleware.UserInfo{ID: other, Roles: []string{"user"}}
	staffUser := &middleware.UserInfo{ID: staff, Roles: []string{"moderator"}}

	cases := []struct {
		name  string
		topic *model.Topic
		user  *middleware.UserInfo
		want  replyCaps
	}{
		{"anonymous", published, nil, replyCaps{}},
		{"author", published, authorUser, replyCaps{Edit: true, Delete: true}},
		{"other", published, otherUser, replyCaps{Like: true}},
		{"staff", published, staffUser, replyCaps{Edit: true, Delete: true, Like: true}},
		{"other on hidden topic", hidden, otherUser, replyCaps{}},
		{"author on hidden topic", hidden, authorUser, replyCaps{Edit: true, Delete: true}},
	}
	for _, c := range cases {
		if got := capsForReply(c.topic, reply, c.user); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
