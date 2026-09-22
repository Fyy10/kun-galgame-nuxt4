package service

import (
	"testing"

	"kun-galgame-api/internal/middleware"
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
