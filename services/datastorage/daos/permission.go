package daos

import (
	"done/services/datastorage/models"
	"done/tools/collections"

	"done/services/datastorage/daos/scopes"
)

// IPermissionRepository its data access layer abstraction of permission.
type IPermission interface {
	Migratable

	// single fetch options

	GetPermissionByID(ID uint) (permission models.Permission, err error)
	GetPermissionByGuardName(guardName string) (permission models.Permission, err error)

	// Multiple fetch options

	GetPermissions(IDs []uint) (permissions collections.Permission, err error)
	GetPermissionsByGuardNames(guardNames []string) (permissions collections.Permission, err error)

	// ID fetch options

	GetPermissionIDs(pagination scopes.GormPager) (permissionIDs []uint, totalCount int64, err error)
	GetDirectPermissionIDsOfUserByID(userID uint, pagination scopes.GormPager) (permissionIDs []uint, totalCount int64, err error)
	GetPermissionIDsOfRolesByIDs(roleIDs []uint, pagination scopes.GormPager) (permissionIDs []uint, totalCount int64, err error)

	// FirstOrCreate & Updates & Delete

	FirstOrCreate(permission *models.Permission) (err error)
	Updates(permission *models.Permission, updates map[string]interface{}) (err error)
	Delete(permission *models.Permission) (err error)
}

// GetPermissionByID get permission by id.
// @param uint
// @return models.Permission, error
func (dao *Dao) GetPermissionByID(ID uint) (permission models.Permission, err error) {
	err = dao.Database.First(&permission, "permissions.id = ?", ID).Error
	return
}

// GetPermissionByGuardName get permission by guard name.
// @param string
// @return models.Permission, error
func (dao *Dao) GetPermissionByGuardName(guardName string) (permission models.Permission, err error) {
	err = dao.Database.Where("permissions.guard_name = ?", guardName).First(&permission).Error
	return
}

// MULTIPLE FETCH OPTIONS

// GetPermissions get permissions by ids.
// @param []uint
// @return collections.Role, error
func (dao *Dao) GetPermissions(IDs []uint) (permissions collections.Permission, err error) {
	err = dao.Database.Where("permissions.id IN (?)", IDs).Find(&permissions).Error
	return
}

// GetPermissionsByGuardNames get permissions by guard names.
// @param []string
// @return collections.Permission, error
func (dao *Dao) GetPermissionsByGuardNames(guardNames []string) (permissions collections.Permission, err error) {
	err = dao.Database.Where("permissions.guard_name IN (?)", guardNames).Find(&permissions).Error
	return
}

// ID FETCH OPTIONS

// GetPermissionIDs get permission ids. (with pagination)
// @param repositories_scopes.GormPager
// @return []uint, int64, error
func (dao *Dao) GetPermissionIDs(pagination scopes.GormPager) (permissionIDs []uint, totalCount int64, err error) {
	err = dao.Database.Model(&models.Permission{}).Count(&totalCount).Scopes(dao.paginate(pagination)).Pluck("permissions.id", &permissionIDs).Error
	return
}

// GetDirectPermissionIDsOfUserByID get direct permission ids of user. (with pagination)
// @param uint
// @param repositories_scopes.GormPager
// @return []uint, int64, error
func (dao *Dao) GetDirectPermissionIDsOfUserByID(userID uint, pagination scopes.GormPager) (permissionIDs []uint, totalCount int64, err error) {
	err = dao.Database.Table("user_permissions").Where("user_permissions.user_id = ?", userID).Count(&totalCount).Scopes(dao.paginate(pagination)).Pluck("user_permissions.permission_id", &permissionIDs).Error
	return
}

// GetPermissionIDsOfRolesByIDs get permission ids of roles. (with pagination)
// @param []uint
// @param repositories_scopes.GormPager
// @return []uint, int64, error
func (dao *Dao) GetPermissionIDsOfRolesByIDs(roleIDs []uint, pagination scopes.GormPager) (permissionIDs []uint, totalCount int64, err error) {
	err = dao.Database.Table("role_permissions").Distinct("role_permissions.permission_id").Where("role_permissions.role_id IN (?)", roleIDs).Count(&totalCount).Scopes(dao.paginate(pagination)).Pluck("role_permissions.permission_id", &permissionIDs).Error
	return
}
