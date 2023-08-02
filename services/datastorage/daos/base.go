package daos

import (
	"done/services/datastorage/daos/scopes"
	"done/services/datastorage/models"
	"done/services/datastorage/models/pivot"

	"gorm.io/gorm"
)

// Seedable gives models the ability to seed.
type Seedable interface {
	Seed() error
}

// Seeds seed to seedable models.
func Seeds(repos ...Seedable) (err error) {
	for _, r := range repos {
		err = r.Seed()
	}
	return
}

// Migratable gives models the ability to migrate.
type Migratable interface {
	Migrate() error
}

// Migrates migrate to migratable models.
func Migrates(daos ...Migratable) (err error) {
	for _, r := range daos {
		err = r.Migrate()
	}
	return
}

// PermissionRepository its data access layer of permission.
type Dao struct {
	Database *gorm.DB
}

// Migrate generate tables from the database.
// @return error
func (dao *Dao) Migrate() {
	dao.Database.AutoMigrate(models.Permission{})
	dao.Database.AutoMigrate(pivot.UserPermissions{})
	dao.Database.AutoMigrate(models.Role{})
	dao.Database.AutoMigrate(pivot.UserRoles{})
	dao.Database.AutoMigrate(models.Role{})
	dao.Database.AutoMigrate(pivot.UserRoles{})
	// TODO err catch
}

// FirstOrCreate & Updates & Delete

// FirstOrCreate create new role if name not exist.
// @param *models.Role
// @return error
func (dao *Dao) FirstOrCreate(m *models.BaseModel) error {
	return nil
}

// Updates update role.
// @param *models.Role
// @param map[string]interface{}
// @return error
func (dao *Dao) Updates(m *models.BaseModel, updates map[string]interface{}) (err error) {
	return dao.Database.Model(m).Updates(updates).Error
}

// Delete delete role.
// @param *models.Role
// @return error
func (dao *Dao) Delete(m *models.BaseModel) (err error) {
	return dao.Database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_roles.role_id = ?", m.ID).Delete(&pivot.UserRoles{}).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Delete(m).Error; err != nil {
			tx.Rollback()
			return err
		}
		return nil
	})
}

// paginate pagging if pagination option is true.
// @param repositories_scopes.GormPager
// @return func(db *gorm.DB) *gorm.DB
func (dao *Dao) paginate(pagination scopes.GormPager) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if pagination != nil {
			db.Scopes(pagination.ToPaginate())
		}

		return db
	}
}
