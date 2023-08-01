package dal

import (
	"context"

	dl "done/services/auth/dal"
	"done/services/menu/sch"
	"done/tools/errors"
	"done/util"

	"gorm.io/gorm"
)

// Get menu storage instance
func GetMenuDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(sch.Menu))
}

// Menu management for RBAC
type Menu struct {
	DB *gorm.DB
}

// Query menus from the database based on the provided parameters and options.
func (a *Menu) Query(ctx context.Context, params sch.MenuQueryParam, opts ...sch.MenuQueryOptions) (*sch.MenuQueryResult, error) {
	var opt sch.MenuQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetMenuDB(ctx, a.DB)

	if v := params.InIDs; len(v) > 0 {
		db = db.Where("id IN ?", v)
	}
	if v := params.LikeName; len(v) > 0 {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}
	if v := params.Status; len(v) > 0 {
		db = db.Where("status = ?", v)
	}
	if v := params.ParentID; len(v) > 0 {
		db = db.Where("parent_id = ?", v)
	}
	if v := params.ParentPathPrefix; len(v) > 0 {
		db = db.Where("parent_path LIKE ?", v+"%")
	}
	if v := params.UserID; len(v) > 0 {
		userRoleQuery := dl.GetUserRoleDB(ctx, a.DB).Where("user_id = ?", v).Select("role_id")
		roleMenuQuery := dl.GetRoleMenuDB(ctx, a.DB).Where("role_id IN (?)", userRoleQuery).Select("menu_id")
		db = db.Where("id IN (?)", roleMenuQuery)
	}
	if v := params.RoleID; len(v) > 0 {
		roleMenuQuery := dl.GetRoleMenuDB(ctx, a.DB).Where("role_id = ?", v).Select("menu_id")
		db = db.Where("id IN (?)", roleMenuQuery)
	}

	var list sch.Menus
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &sch.MenuQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified menu from the database.
func (a *Menu) Get(ctx context.Context, id string, opts ...sch.MenuQueryOptions) (*sch.Menu, error) {
	var opt sch.MenuQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(sch.Menu)
	ok, err := util.FindOne(ctx, GetMenuDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Checks if the specified menu exists in the database.
func (a *Menu) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetMenuDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// Checks if the specified menu code exists under the specified parent ID in the database.
func (a *Menu) ExistsCodeByParentID(ctx context.Context, parentID, code string) (bool, error) {
	ok, err := util.Exists(ctx, GetMenuDB(ctx, a.DB).Where("parent_id=? AND code=?", parentID, code))
	return ok, errors.WithStack(err)
}

// Create a new menu.
func (a *Menu) Create(ctx context.Context, item *sch.Menu) error {
	result := GetMenuDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified menu in the database.
func (a *Menu) Update(ctx context.Context, item *sch.Menu) error {
	result := GetMenuDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified menu from the database.
func (a *Menu) Delete(ctx context.Context, id string) error {
	result := GetMenuDB(ctx, a.DB).Where("id=?", id).Delete(new(sch.Menu))
	return errors.WithStack(result.Error)
}

// Updates the parent path of the specified menu.
func (a *Menu) UpdateParentPath(ctx context.Context, id, parentPath string) error {
	result := GetMenuDB(ctx, a.DB).Where("id=?", id).Update("parent_path", parentPath)
	return errors.WithStack(result.Error)
}

// Updates the status of all menus whose parent path starts with the provided parent path.
func (a *Menu) UpdateStatusByParentPath(ctx context.Context, parentPath, status string) error {
	result := GetMenuDB(ctx, a.DB).Where("parent_path like ?", parentPath+"%").Update("status", status)
	return errors.WithStack(result.Error)
}
