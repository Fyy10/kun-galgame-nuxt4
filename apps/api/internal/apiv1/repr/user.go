package repr

import "kun-galgame-api/pkg/userclient"

type UserRef struct {
	Object string    `json:"object" enum:"user" maxLength:"4" doc:"Type discriminant. Always user."`
	ID     DecimalID `json:"id" doc:"User id. JSON string of a decimal integer."`
	Name   *string   `json:"name" maxLength:"64" doc:"Display name. null when the account no longer exists; show a localized label. Free text; never use it as a decision input."`
	Avatar *Image    `json:"avatar" doc:"Avatar image. null when the account has no image-service hash."`
}

func NewUserRef(cdnBase string, u userclient.User) UserRef {
	name := u.Name
	return UserRef{
		Object: "user",
		ID:     ID(u.ID),
		Name:   &name,
		Avatar: NewImage(cdnBase, u.AvatarImageHash, nil),
	}
}

// userclient.Placeholder gives a missing account a fixed Chinese display name,
// which a client that localizes cannot tell from a user who picked that name.
func DeletedUserRef(id int) UserRef {
	return UserRef{Object: "user", ID: ID(id)}
}
