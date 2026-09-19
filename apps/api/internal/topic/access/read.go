package access

import (
	"slices"
	"strconv"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/perm"
)

type View struct {
	ID             int
	Roles          []string
	Authenticated  bool
	ViewHidden     bool
	ViewRestricted bool
}

func Snapshot(user *middleware.UserInfo) View {
	if user == nil {
		return View{}
	}
	return View{
		ID:             user.ID,
		Roles:          user.Roles,
		Authenticated:  true,
		ViewHidden:     user.Can(perm.TopicViewHidden),
		ViewRestricted: user.Can(perm.TopicViewRestricted),
	}
}

func NeedsGrants(topic *model.Topic) bool {
	return topic != nil && (topic.AccessScope == "role" || topic.AccessScope == "users")
}

func CanRead(topic *model.Topic, user *middleware.UserInfo, grants []model.TopicAccessGrant) bool {
	return Allowed(topic, Snapshot(user), grants)
}

func Allowed(topic *model.Topic, viewer View, grants []model.TopicAccessGrant) bool {
	if topic == nil {
		return false
	}
	author := viewer.Authenticated && viewer.ID == topic.UserID
	if topic.Status == 1 && !author && !(viewer.Authenticated && viewer.ViewHidden) {
		return false
	}
	if author || (viewer.Authenticated && viewer.ViewRestricted) {
		return true
	}
	switch topic.AccessScope {
	case "public":
		return true
	case "login":
		return viewer.Authenticated
	case "role", "users":
		if !viewer.Authenticated {
			return false
		}
		for _, grant := range grants {
			if topic.AccessScope == "role" && grant.SubjectType == "role" && slices.Contains(viewer.Roles, grant.SubjectValue) {
				return true
			}
			if topic.AccessScope == "users" && grant.SubjectType == "user" && grant.SubjectValue == strconv.Itoa(viewer.ID) {
				return true
			}
		}
	}
	return false
}
