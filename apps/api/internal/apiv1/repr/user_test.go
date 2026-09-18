package repr

import (
	"encoding/json"
	"testing"

	"kun-galgame-api/pkg/userclient"
)

func TestNewUserRefAvatarFromHash(t *testing.T) {
	u := userclient.User{ID: 3, Name: "alice", AvatarImageHash: testHash, Avatar: "https://ignored.example/a.png"}
	ref := NewUserRef("https://cdn", u)
	if ref.Object != "user" || string(ref.ID) != "3" || ref.Name != "alice" {
		t.Errorf("ref %+v", ref)
	}
	if ref.Avatar == nil || ref.Avatar.Hash != testHash {
		t.Errorf("avatar %+v", ref.Avatar)
	}
	if ref.Avatar.Width != nil || ref.Avatar.Sexual != nil {
		t.Errorf("hash-only avatar must have null meta: %+v", ref.Avatar)
	}
}

func TestNewUserRefExternalURLIsNullAvatar(t *testing.T) {
	u := userclient.User{ID: 8, Name: "hotlink", Avatar: "https://i0.hdslb.com/face.png", AvatarImageHash: ""}
	ref := NewUserRef("https://cdn", u)
	if ref.Avatar != nil {
		t.Errorf("external URL avatar = %+v, want null", ref.Avatar)
	}
	raw, _ := json.Marshal(ref)
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["avatar"] != nil {
		t.Errorf("avatar marshalled %s", raw)
	}
}

func TestNewUserRefPlaceholderPassesThrough(t *testing.T) {
	u := userclient.Placeholder(99)
	ref := NewUserRef("https://cdn", u)
	if ref.Name != "已注销用户" || string(ref.ID) != "99" || ref.Avatar != nil {
		t.Errorf("placeholder %+v", ref)
	}
}
