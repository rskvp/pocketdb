package options

import "done/services/datastorage/utils"

// RoleOption represents options when fetching roles.
type RoleOption struct {
	WithPermissions bool
	Pagination      *utils.Pagination
}
