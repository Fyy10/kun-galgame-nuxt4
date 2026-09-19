package service

import (
	"encoding/json"
	"strings"
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
)

func TestRequireTopicReadWithoutGrants(t *testing.T) {
	for _, tt := range []struct {
		name, scope string
		status      int
		viewer      *middleware.UserInfo
		denied      bool
	}{
		{"public anonymous", "public", 0, nil, false},
		{"login anonymous", "login", 0, nil, true},
		{"login viewer", "login", 0, &middleware.UserInfo{ID: 2}, false},
		{"hidden public", "public", 1, nil, true},
		{"hidden role", "role", 1, nil, true},
		{"hidden users", "users", 1, nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := requireTopicRead(nil, &topicModel.Topic{UserID: 1, AccessScope: tt.scope, Status: tt.status}, tt.viewer)
			if (err != nil) != tt.denied {
				t.Fatalf("error = %v", err)
			}
			if err != nil && err.StatusCode != 404 {
				t.Fatalf("status = %d", err.StatusCode)
			}
			if err != nil && err.Message != "未找到该话题" {
				t.Fatalf("message = %q", err.Message)
			}
		})
	}
}

func TestTopicDetailGrantsPrivacy(t *testing.T) {
	topic := &topicModel.Topic{UserID: 1, AccessScope: "users"}
	grants := []topicModel.TopicAccessGrant{{SubjectType: "user", SubjectValue: "2"}, {SubjectType: "role", SubjectValue: "creator"}}
	for _, tt := range []struct {
		name    string
		viewer  *middleware.UserInfo
		visible bool
	}{
		{"anonymous", nil, false},
		{"granted viewer", &middleware.UserInfo{ID: 2, Roles: []string{"creator"}}, false},
		{"author", &middleware.UserInfo{ID: 1}, true},
		{"editor", &middleware.UserInfo{ID: 3, Roles: []string{"admin"}}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			detail := dto.TopicDetail{AccessScope: topic.AccessScope, AccessGrants: topicDetailGrants(topic, tt.viewer, grants)}
			data, err := json.Marshal(detail)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), `"access_grants"`) != tt.visible {
				t.Fatalf("unexpected grants visibility: %s", data)
			}
			if !strings.Contains(string(data), `"access_scope":"users"`) {
				t.Fatal("missing access_scope")
			}
			if tt.visible && (len(detail.AccessGrants.UserIDs) != 1 || detail.AccessGrants.UserIDs[0] != 2 || len(detail.AccessGrants.Roles) != 1) {
				t.Fatalf("grants = %+v", detail.AccessGrants)
			}
		})
	}
}
