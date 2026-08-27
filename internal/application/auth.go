package application

import "blast-permit/internal/domain"

const (
	RoleDesigner      = "designer"
	RoleReviewer      = "reviewer"
	RoleSafetyOfficer = "safety_officer"
)

type Actor struct {
	Role string
	Name string
}

func requireRole(a Actor, roles ...string) error {
	if a.Name == "" {
		return domain.NewError(domain.CodeForbidden, "必须提供操作者姓名")
	}
	for _, r := range roles {
		if a.Role == r {
			return nil
		}
	}
	return domain.NewError(domain.CodeForbidden, "角色 %q 无权执行此操作", a.Role)
}
