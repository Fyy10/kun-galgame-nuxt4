package role

import "slices"

const (
	Creator   = "creator"
	Moderator = "moderator"
	Admin     = "admin"
	Ren       = "ren"
)

func Has(roles []string, name string) bool {
	return slices.Contains(roles, name)
}

func CanModerate(roles []string) bool {
	return Has(roles, Moderator) || Has(roles, Admin) || Has(roles, Ren)
}

func CanAdminister(roles []string) bool {
	return Has(roles, Admin) || Has(roles, Ren)
}

func IsCreator(roles []string) bool {
	return Has(roles, Creator)
}

func WithoutStaff(roles []string) []string {
	return slices.DeleteFunc(slices.Clone(roles), func(r string) bool {
		return r == Moderator || r == Admin || r == Ren
	})
}

func Union(roles, siteRoles []string) []string {
	if len(siteRoles) == 0 {
		return roles
	}
	out := slices.Clone(roles)
	for _, r := range siteRoles {
		if !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	return out
}
