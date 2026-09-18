package repr

import "kun-galgame-api/pkg/userclient"

type UserRef struct {
	Object string    `json:"object" enum:"user" maxLength:"4" doc:"Type discriminant. Always user."`
	ID     DecimalID `json:"id" doc:"User id. JSON string of a decimal integer."`
	Name   string    `json:"name" maxLength:"64" doc:"Display name. Free text; never use it as a decision input."`
	Avatar *Image    `json:"avatar" doc:"Avatar image. null when the account has no image-service hash."`
}

func NewUserRef(cdnBase string, u userclient.User) UserRef {
	return UserRef{
		Object: "user",
		ID:     ID(u.ID),
		Name:   u.Name,
		Avatar: NewImage(cdnBase, u.AvatarImageHash, nil),
	}
}
