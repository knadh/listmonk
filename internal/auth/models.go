package auth

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

var ErrPermDenied = echo.NewHTTPError(http.StatusForbidden, "permission denied")

// PermType indicates a generic permission type which is either get (read) or manage (write).
type PermType uint8

const (
	PermTypeGet PermType = 1 << iota
	PermTypeManage
)

const (
	// UserKey is the key on which the User profile is set on echo handlers.
	UserKey    = "auth_user"
	SessionKey = "auth_session"
)

const (
	// SuperAdminRoleID is the database ID of the primordial super admin role.
	SuperAdminRoleID = 1

	// User.
	UserTypeUser       = "user"
	UserTypeAPI        = "api"
	UserStatusEnabled  = "enabled"
	UserStatusDisabled = "disabled"

	// Role.
	RoleTypeUser = "user"
	RoleTypeList = "list"
)

// List of all granular permissions.
const (
	PermListGetAll            = "lists:get_all"
	PermListManageAll         = "lists:manage_all"
	PermListManage            = "list:manage"
	PermListGet               = "list:get"
	PermSubscribersGet        = "subscribers:get"
	PermSubscribersGetAll     = "subscribers:get_all"
	PermSubscribersManage     = "subscribers:manage"
	PermSubscribersImport     = "subscribers:import"
	PermSubscribersSqlQuery   = "subscribers:sql_query"
	PermTxSend                = "tx:send"
	PermCampaignsGet          = "campaigns:get"
	PermCampaignsGetAll       = "campaigns:get_all"
	PermCampaignsGetAnalytics = "campaigns:get_analytics"
	PermCampaignsManage       = "campaigns:manage"
	PermCampaignsManageAll    = "campaigns:manage_all"
	PermBouncesGet            = "bounces:get"
	PermBouncesManage         = "bounces:manage"
	PermWebhooksPostBounce    = "webhooks:post_bounce"
	PermMediaGet              = "media:get"
	PermMediaManage           = "media:manage"
	PermTemplatesGet          = "templates:get"
	PermTemplatesManage       = "templates:manage"
	PermUsersGet              = "users:get"
	PermUsersManage           = "users:manage"
	PermRolesGet              = "roles:get"
	PermRolesManage           = "roles:manage"
	PermSettingsGet           = "settings:get"
	PermSettingsManage        = "settings:manage"
	PermSettingsMaintain      = "settings:maintain"
)

// Base holds common fields shared across models.
type Base struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt null.Time `db:"created_at" json:"created_at"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at"`
}

// User represents an admin user.
type User struct {
	Base

	Username string `db:"username" json:"username"`

	// For API users, this is the plaintext API token.
	Password null.String `db:"password" json:"password,omitempty"`

	PasswordLogin bool             `db:"password_login" json:"password_login"`
	Email         null.String      `db:"email" json:"email"`
	Name          string           `db:"name" json:"name"`
	Type          string           `db:"type" json:"type"`
	Status        string           `db:"status" json:"status"`
	Avatar        null.String      `db:"avatar" json:"avatar"`
	LoggedInAt    null.Time        `db:"loggedin_at" json:"loggedin_at"`
	UserRoleID    int              `db:"user_role_id" json:"user_role_id,omitempty"`
	UserRoleName  string           `db:"user_role_name" json:"-"`
	ListRoleID    *int             `db:"list_role_id" json:"list_role_id,omitempty"`
	ListRoleName  null.String      `db:"list_role_name" json:"-"`
	UserRolePerms pq.StringArray   `db:"user_role_permissions" json:"-"`
	ListsPermsRaw *json.RawMessage `db:"list_role_perms" json:"-"`

	// Non-DB fields filled post-retrieval.
	UserRole struct {
		ID          int      `db:"-" json:"id"`
		Name        string   `db:"-" json:"name"`
		Permissions []string `db:"-" json:"permissions"`
	} `db:"-" json:"user_role"`

	ListRole           *ListRolePermissions        `db:"-" json:"list_role"`
	PermissionsMap     map[string]struct{}         `db:"-" json:"-"`
	ListPermissionsMap map[int]map[string]struct{} `db:"-" json:"-"`
	GetListIDs         []int                       `db:"-" json:"-"`
	ManageListIDs      []int                       `db:"-" json:"-"`
	HasPassword        bool                        `db:"-" json:"-"`
}

type ListPermission struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Permissions pq.StringArray `json:"permissions"`
}

type ListRolePermissions struct {
	ID    int              `db:"-" json:"id"`
	Name  string           `db:"-" json:"name"`
	Lists []ListPermission `db:"-" json:"lists"`
}

type Role struct {
	Base

	Type        string         `db:"type" json:"type"`
	Name        null.String    `db:"name" json:"name"`
	Permissions pq.StringArray `db:"permissions" json:"permissions"`

	ListID   null.Int         `db:"list_id" json:"-"`
	ParentID null.Int         `db:"parent_id" json:"-"`
	ListsRaw json.RawMessage  `db:"list_permissions" json:"-"`
	Lists    []ListPermission `db:"-" json:"lists"`
}

type ListRole struct {
	Base

	Name null.String `db:"name" json:"name"`

	ListID   null.Int         `db:"list_id" json:"-"`
	ParentID null.Int         `db:"parent_id" json:"-"`
	ListsRaw json.RawMessage  `db:"list_permissions" json:"-"`
	Lists    []ListPermission `db:"-" json:"lists"`
}

// HasPerm checks if the user has a specific permission.
func (u *User) HasPerm(perm string) bool {
	// Short-circuit if the user is the primordial super admin.
	if u.UserRoleID == SuperAdminRoleID {
		return true
	}

	_, ok := u.PermissionsMap[perm]
	return ok
}

// HasListPerm checks if the user has get or manage access to the given list.
// perm is either PermGet or PermManage.
func (u *User) HasListPerm(types PermType, listIDs ...int) error {
	var permAll, perm string

	if types == 0 {
		return ErrPermDenied
	}

	if types&PermTypeGet != 0 {
		permAll = PermListGetAll
		perm = PermListGet
	} else if types&PermTypeManage != 0 {
		permAll = PermListManageAll
		perm = PermListManage
	}

	// Check if the user has permissions for all lists or the specific list.
	if u.HasPerm(permAll) {
		return nil
	}

	for _, id := range listIDs {
		if id > 0 {
			if u.hasListPerm(perm, id) {
				return nil
			}
		}
	}

	return ErrPermDenied
}

func (u *User) hasListPerm(perm string, listID int) bool {
	// Short-circuit if the user is the primordial super admin.
	if u.UserRoleID == SuperAdminRoleID {
		return true
	}

	if _, ok := u.ListPermissionsMap[listID]; !ok {
		return false
	}

	_, ok := u.ListPermissionsMap[listID][perm]
	return ok
}

// GetPermittedLists returns a list of IDs the user has access to based on
// the given get / manage permissions. If the user has the blanket "*_all"
// permission (or the user is a super admin), then the bool is set to true and
// the list is nil as all lists are permitted.
func (u *User) GetPermittedLists(types PermType) (bool, []int) {
	if types == 0 {
		return false, nil
	}

	// Short-circuit if the user is the primordial super admin.
	if u.UserRoleID == SuperAdminRoleID {
		return true, nil
	}

	var (
		get    = types&PermTypeGet != 0
		manage = types&PermTypeManage != 0
	)

	// If the user has the list:get_all or list:manage_all permission, no
	// further checks are required.
	if get {
		if _, ok := u.PermissionsMap[PermListGetAll]; ok {
			return true, nil
		}
	}
	if manage {
		if _, ok := u.PermissionsMap[PermListManageAll]; ok {
			return true, nil
		}
	}

	if get {
		// If the user has per-list permissions, return that. Otherwise, let the
		// 'manage' permission check run.
		if len(u.GetListIDs) > 0 {
			out := make([]int, len(u.GetListIDs))
			copy(out, u.GetListIDs)
			return false, out
		}
	}

	if manage {
		// User has per-list permissions.
		out := make([]int, len(u.ManageListIDs))
		copy(out, u.ManageListIDs)
		return false, out
	}

	return false, nil
}

// FilterListsByPerm returns list IDs filtered by either of the given perms.
func (u *User) FilterListsByPerm(types PermType, listIDs []int) []int {
	if types == 0 {
		return nil
	}

	var (
		get    = types&PermTypeGet != 0
		manage = types&PermTypeManage != 0
	)

	// If the user has full list management permission,
	// no further checks are required.
	if get {
		if _, ok := u.PermissionsMap[PermListGetAll]; ok {
			return listIDs
		}
	}
	if manage {
		if _, ok := u.PermissionsMap[PermListManageAll]; ok {
			return listIDs
		}
	}

	out := make([]int, 0, len(listIDs))
	for _, id := range listIDs {
		// Check if it exists in the map.
		if l, ok := u.ListPermissionsMap[id]; ok {
			// Check if any of the given permission exists for it.
			if get {
				if _, ok := l[PermListGet]; ok {
					out = append(out, id)
				}
			} else if manage {
				if _, ok := l[PermListManage]; ok {
					out = append(out, id)
				}
			}
		}
	}

	return out
}
