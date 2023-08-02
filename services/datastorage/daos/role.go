package daos

import (
	"done/services/datastorage/models"
	"done/tools/collections"

	"done/services/datastorage/daos/scopes"
)

// IRoleRepository its data access layer abstraction of role.
type IRole interface {
	Migratable

	// single fetch options

	GetRoleByID(ID uint) (role models.Role, err error)
	GetRoleByIDWithPermissions(ID uint) (role models.Role, err error)

	GetRoleByGuardName(guardName string) (role models.Role, err error)
	GetRoleByGuardNameWithPermissions(guardName string) (role models.Role, err error)

	// Multiple fetch options

	GetRoles(roleIDs []uint) (roles collections.Role, err error)
	GetRolesWithPermissions(roleIDs []uint) (roles collections.Role, err error)

	GetRolesByGuardNames(guardNames []string) (roles collections.Role, err error)
	GetRolesByGuardNamesWithPermissions(guardNames []string) (roles collections.Role, err error)

	// ID fetch options

	GetRoleIDs(pagination scopes.GormPager) (roleIDs []uint, totalCount int64, err error)
	GetRoleIDsOfUser(userID uint, pagination scopes.GormPager) (roleIDs []uint, totalCount int64, err error)
	GetRoleIDsOfPermission(permissionID uint, pagination scopes.GormPager) (roleIDs []uint, totalCount int64, err error)

	// FirstOrCreate & Updates & Delete

	FirstOrCreate(role *models.Role) (err error)
	Updates(role *models.Role, updates map[string]interface{}) (err error)
	Delete(role *models.Role) (err error)

	// Actions

	AddPermissions(role *models.Role, permissions collections.Permission) (err error)
	ReplacePermissions(role *models.Role, permissions collections.Permission) (err error)
	RemovePermissions(role *models.Role, permissions collections.Permission) (err error)
	ClearPermissions(role *models.Role) (err error)

	// Controls

	HasPermission(roles collections.Role, permission models.Permission) (b bool, err error)
	HasAllPermissions(roles collections.Role, permissions collections.Permission) (b bool, err error)
	HasAnyPermissions(roles collections.Role, permissions collections.Permission) (b bool, err error)
}

// SINGLE FETCH OPTIONS

// GetRoleByID get role by id.
// @param uint
// @return models.Role, error
func (dao *Dao) GetRoleByID(ID uint) (role models.Role, err error) {
	err = dao.Database.First(&role, "roles.id = ?", ID).Error
	return
}

// GetRoleByIDWithPermissions get role by id with its permissions.
// @param uint
// @return models.Role, error
func (dao *Dao) GetRoleByIDWithPermissions(ID uint) (role models.Role, err error) {
	err = dao.Database.Preload("Permissions").First(&role, "roles.id = ?", ID).Error
	return
}

// GetRoleByGuardName get role by guard name.
// @param string
// @return models.Role, error
func (dao *Dao) GetRoleByGuardName(guardName string) (role models.Role, err error) {
	err = dao.Database.Where("roles.guard_name = ?", guardName).First(&role).Error
	return
}

// GetRoleByGuardNameWithPermissions get role by guard name with its permissions.
// @param string
// @return models.Role, error
func (dao *Dao) GetRoleByGuardNameWithPermissions(guardName string) (role models.Role, err error) {
	err = dao.Database.Preload("Permissions").Where("roles.guard_name = ?", guardName).First(&role).Error
	return
}

// MULTIPLE FETCH OPTIONS

// GetRoles get roles by ids.
// @param []uint
// @return collections.Role, error
func (dao *Dao) GetRoles(IDs []uint) (roles collections.Role, err error) {
	err = dao.Database.Where("roles.id IN (?)", IDs).Find(&roles).Error
	return
}

// GetRolesWithPermissions get roles by ids with its permissions.
// @param []uint
// @return collections.Role, error
func (dao *Dao) GetRolesWithPermissions(IDs []uint) (roles collections.Role, err error) {
	err = dao.Database.Preload("Permissions").Where("roles.id IN (?)", IDs).Find(&roles).Error
	return
}

// GetRolesByGuardNames get roles by guard names.
// @param []string
// @return collections.Role, error
func (dao *Dao) GetRolesByGuardNames(guardNames []string) (roles collections.Role, err error) {
	err = dao.Database.Where("roles.guard_name IN (?)", guardNames).Find(&roles).Error
	return
}

// GetRolesByGuardNamesWithPermissions get roles by guard names.
// @param []string
// @return collections.Role, error
func (dao *Dao) GetRolesByGuardNamesWithPermissions(guardNames []string) (roles collections.Role, err error) {
	err = dao.Database.Preload("Permissions").Where("roles.guard_name IN (?)", guardNames).Find(&roles).Error
	return
}

// ID FETCH OPTIONS

// GetRoleIDs get role ids. (with pagination)
// @param repositories_scopes.GormPager
// @return []uint, int64, error
func (dao *Dao) GetRoleIDs(pagination scopes.GormPager) (roleIDs []uint, totalCount int64, err error) {
	err = dao.Database.Model(&models.Role{}).Count(&totalCount).Scopes(dao.paginate(pagination)).Pluck("roles.id", &roleIDs).Error
	return
}

// // GetRoleIDsOfUser get role ids of user. (with pagination)
// // @param uint
// // @param repositories_scopes.GormPager
// // @return []uint, int64, error
// func (dao *Dao) GetRoleIDsOfUser(userID uint, pagination scopes.GormPager) (roleIDs []uint, totalCount int64, err error) {
// 	err = dao.Database.Table("user_roles").Where("user_roles.user_id = ?", userID).Count(&totalCount).Scopes(dao.paginate(pagination)).Pluck("user_roles.role_id", &roleIDs).Error
// 	return
// }

// GetRoleIDsOfPermission get role ids of permission. (with pagination)
// @param uint
// @param repositories_scopes.GormPager
// @return []uint, int64, error
func (dao *Dao) GetRoleIDsOfPermission(permissionID uint, pagination scopes.GormPager) (roleIDs []uint, totalCount int64, err error) {
	err = dao.Database.Table("role_permissions").Where("role_permissions.permission_id = ?", permissionID).Count(&totalCount).Scopes(dao.paginate(pagination)).Pluck("role_permissions.role_id", &roleIDs).Error
	return
}

// ACTIONS

// AddPermissions add permissions to role.
// @param *models.Role
// @param collections.Permission
// @return error
func (dao *Dao) AddPermissions(role *models.Role, permissions collections.Permission) error {
	return dao.Database.Model(role).Association("Permissions").Append(permissions.Origin())
}

// ReplacePermissions replace permissions of role.
// @param *models.Role
// @param collections.Permission
// @return error
func (dao *Dao) ReplacePermissions(role *models.Role, permissions collections.Permission) error {
	return dao.Database.Model(role).Association("Permissions").Replace(permissions.Origin())
}

// RemovePermissions remove permissions of role.
// @param *models.Role
// @param collections.Permission
// @return error
func (dao *Dao) RemovePermissions(role *models.Role, permissions collections.Permission) error {
	return dao.Database.Model(role).Association("Permissions").Delete(permissions.Origin())
}

// ClearPermissions remove all permissions of role.
// @param *models.Role
// @return error
func (dao *Dao) ClearPermissions(role *models.Role) (err error) {
	return dao.Database.Model(role).Association("Permissions").Clear()
}

// Controls

// HasPermission does the role or any of the roles have given permission?
// @param collections.Role
// @param models.Permission
// @return bool, error
func (dao *Dao) HasPermission(roles collections.Role, permission models.Permission) (b bool, err error) {
	var count int64
	err = dao.Database.Table("role_permissions").Where("role_permissions.role_id IN (?)", roles.IDs()).Where("role_permissions.permission_id = ?", permission.ID).Count(&count).Error
	return count > 0, err
}

// HasAllPermissions does the role or roles have all the given permissions?
// @param collections.Role
// @param collections.Permission
// @return bool, error
func (dao *Dao) HasAllPermissions(roles collections.Role, permissions collections.Permission) (b bool, err error) {
	var count int64
	err = dao.Database.Table("role_permissions").Where("role_permissions.role_id IN (?)", roles.IDs()).Where("role_permissions.permission_id IN (?)", permissions.IDs()).Count(&count).Error
	return roles.Len()*permissions.Len() == count, err
}

// HasAnyPermissions does the role or roles have any of the given permissions?
// @param collections.Role
// @param collections.Permission
// @return bool, error
func (dao *Dao) HasAnyPermissions(roles collections.Role, permissions collections.Permission) (b bool, err error) {
	var count int64
	err = dao.Database.Table("role_permissions").Where("role_permissions.role_id IN (?)", roles.IDs()).Where("role_permissions.permission_id IN (?)", permissions.IDs()).Count(&count).Error
	return count > 0, err
}
