package handler

import (
	"testing"

	adminModel "kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/middleware"
)

func TestCanEndClaimedTodo(t *testing.T) {
	claimer := 101
	claimed := &adminModel.Todo{ID: 7, Status: adminModel.TodoStatusClaimed, ClaimedUserID: &claimer}
	unclaimed := &adminModel.Todo{ID: 8, Status: adminModel.TodoStatusClaimed}

	cases := []struct {
		name  string
		todo  *adminModel.Todo
		uid   int
		roles []string
		want  bool
	}{
		{"claimer", claimed, claimer, []string{"user"}, true},
		{"another editor", claimed, 202, []string{"moderator"}, true},
		{"admin", claimed, 202, []string{"admin"}, true},
		{"stranger", claimed, 202, []string{"user"}, false},
		{"editor on a row with no claimer", unclaimed, 202, []string{"moderator"}, true},
		{"stranger on a row with no claimer", unclaimed, 202, []string{"user"}, false},
	}
	for _, c := range cases {
		if got := canEndClaimedTodo(c.todo, &middleware.UserInfo{ID: c.uid, Roles: c.roles}); got != c.want {
			t.Errorf("%s: canEndClaimedTodo = %v, want %v", c.name, got, c.want)
		}
	}
}
