package main

import (
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/labstack/echo/v4"
)

type adminView struct {
	Title       string
	Description string
	Profile     auth.User
}

func newAdminView(c echo.Context, title, description string) adminView {
	return adminView{
		Title:       title,
		Description: description,
		Profile:     auth.GetUser(c),
	}
}

func (v adminView) Can(perms ...string) bool {
	return can(v.Profile, perms...)
}

func (v adminView) CanManageList(id int) bool {
	return canManageList(v.Profile, id)
}

func can(user auth.User, perms ...string) bool {
	for _, perm := range perms {
		if before, ok := strings.CutSuffix(perm, "*"); ok {
			prefix := before
			for _, p := range user.UserRole.Permissions {
				if strings.HasPrefix(p, prefix) {
					return true
				}
			}
			continue
		}
		if user.HasPerm(perm) {
			return true
		}
	}
	return false
}

func canManageList(user auth.User, id int) bool {
	return user.HasListPerm(auth.PermTypeManage, id) == nil
}
