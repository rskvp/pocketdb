package biz

import (
	"context"
	"sort"
	"time"

	"done/services/auth/dal"
	"done/services/auth/sch"
	"done/tools/cachex"
	"done/tools/config"
	"done/tools/crypto/hash"
	"done/tools/errors"
	"done/tools/jwtx"
	"done/tools/logging"
	"done/util"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Login management for RBAC
type Login struct {
	Cache       cachex.Cacher
	Auth        jwtx.Auther
	UserDAL     *dal.User
	UserRoleDAL *dal.UserRole
	MenuDAL     *dal.Menu
	UserBIZ     *User
}

func (a *Login) ParseUserID(c *gin.Context) (string, error) {
	rootID := config.C.General.Root.ID
	if config.C.Middleware.Auth.Disable {
		return rootID, nil
	}

	invalidToken := errors.Unauthorized(config.ErrInvalidTokenID, "Invalid access token")
	token := util.GetToken(c)
	if token == "" {
		return "", invalidToken
	}

	ctx := c.Request.Context()
	ctx = util.NewUserToken(ctx, token)

	userID, err := a.Auth.ParseSubject(ctx, token)
	if err != nil {
		if err == jwtx.ErrInvalidToken {
			return "", invalidToken
		}
		return "", err
	} else if userID == rootID {
		c.Request = c.Request.WithContext(util.NewIsRootUser(ctx))
		return userID, nil
	}

	userCacheVal, ok, err := a.Cache.Get(ctx, config.CacheNSForUser, userID)
	if err != nil {
		return "", err
	} else if ok {
		userCache := util.ParseUserCache(userCacheVal)
		c.Request = c.Request.WithContext(util.NewUserCache(ctx, userCache))
		return userID, nil
	}

	// Check user status, if not activated, force to logout
	user, err := a.UserDAL.Get(ctx, userID, sch.UserQueryOptions{
		QueryOptions: util.QueryOptions{SelectFields: []string{"status"}},
	})
	if err != nil {
		return "", err
	} else if user == nil || user.Status != sch.UserStatusActivated {
		return "", invalidToken
	}

	roleIDs, err := a.UserBIZ.GetRoleIDs(ctx, userID)
	if err != nil {
		return "", err
	}

	userCache := util.UserCache{
		RoleIDs: roleIDs,
	}
	err = a.Cache.Set(ctx, config.CacheNSForUser, userID, userCache.String())
	if err != nil {
		return "", err
	}

	c.Request = c.Request.WithContext(util.NewUserCache(ctx, userCache))
	return userID, nil
}

func (a *Login) genUserToken(ctx context.Context, userID string) (*sch.LoginToken, error) {
	token, err := a.Auth.GenerateToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	tokenBuf, err := token.EncodeToJSON()
	if err != nil {
		return nil, err
	}
	logging.Context(ctx).Info("Generate user token", zap.Any("token", string(tokenBuf)))

	return &sch.LoginToken{
		AccessToken: token.GetAccessToken(),
		TokenType:   token.GetTokenType(),
		ExpiresAt:   token.GetExpiresAt(),
	}, nil
}

func (a *Login) Login(ctx context.Context, formItem *sch.LoginForm) (*sch.LoginToken, error) {

	ctx = logging.NewTag(ctx, logging.TagKeyLogin)

	// login by root
	if formItem.Username == config.C.General.Root.Username {
		if formItem.Password != hash.MD5String(config.C.General.Root.Password) {
			return nil, errors.BadRequest(config.ErrInvalidUsernameOrPassword, "Incorrect password")
		}

		logging.Context(ctx).Info("Login by root")
		return a.genUserToken(ctx, config.C.General.Root.ID)
	}

	// get user info
	user, err := a.UserDAL.GetByUsername(ctx, formItem.Username, sch.UserQueryOptions{
		QueryOptions: util.QueryOptions{
			SelectFields: []string{"id", "password", "status"},
		},
	})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.BadRequest(config.ErrInvalidUsernameOrPassword, "Incorrect username")
	} else if user.Status != sch.UserStatusActivated {
		return nil, errors.BadRequest("", "User status is not activated, please contact the administrator")
	}

	// check password
	if err := hash.CompareHashAndPassword(user.Password, formItem.Password); err != nil {
		return nil, errors.BadRequest(config.ErrInvalidUsernameOrPassword, "Incorrect password")
	}

	userID := user.ID
	ctx = logging.NewUserID(ctx, userID)

	// set user cache with role ids
	roleIDs, err := a.UserBIZ.GetRoleIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	userCache := util.UserCache{RoleIDs: roleIDs}
	err = a.Cache.Set(ctx, config.CacheNSForUser, userID, userCache.String(),
		time.Duration(config.C.Dictionary.UserCacheExp)*time.Hour)
	if err != nil {
		logging.Context(ctx).Error("Failed to set cache", zap.Error(err))
	}
	logging.Context(ctx).Info("Login success", zap.String("username", formItem.Username))

	// generate token
	return a.genUserToken(ctx, userID)
}

func (a *Login) RefreshToken(ctx context.Context) (*sch.LoginToken, error) {
	userID := util.FromUserID(ctx)

	user, err := a.UserDAL.Get(ctx, userID, sch.UserQueryOptions{
		QueryOptions: util.QueryOptions{
			SelectFields: []string{"status"},
		},
	})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.BadRequest("", "Incorrect user")
	} else if user.Status != sch.UserStatusActivated {
		return nil, errors.BadRequest("", "User status is not activated, please contact the administrator")
	}

	return a.genUserToken(ctx, userID)
}

func (a *Login) Logout(ctx context.Context) error {
	userToken := util.FromUserToken(ctx)
	if userToken == "" {
		return nil
	}

	ctx = logging.NewTag(ctx, logging.TagKeyLogout)
	if err := a.Auth.DestroyToken(ctx, userToken); err != nil {
		return err
	}

	userID := util.FromUserID(ctx)
	err := a.Cache.Delete(ctx, config.CacheNSForUser, userID)
	if err != nil {
		logging.Context(ctx).Error("Failed to delete user cache", zap.Error(err))
	}
	logging.Context(ctx).Info("Logout success")

	return nil
}

// Get user info
func (a *Login) GetUserInfo(ctx context.Context) (*sch.User, error) {
	if util.FromIsRootUser(ctx) {
		return &sch.User{
			ID:       config.C.General.Root.ID,
			Username: config.C.General.Root.Username,
			Status:   sch.UserStatusActivated,
		}, nil
	}

	userID := util.FromUserID(ctx)
	user, err := a.UserDAL.Get(ctx, userID, sch.UserQueryOptions{
		QueryOptions: util.QueryOptions{
			OmitFields: []string{"password"},
		},
	})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.NotFound("", "User not found")
	}

	userRoleResult, err := a.UserRoleDAL.Query(ctx, sch.UserRoleQueryParam{
		UserID: userID,
	}, sch.UserRoleQueryOptions{
		JoinRole: true,
	})
	if err != nil {
		return nil, err
	}
	user.Roles = userRoleResult.Data

	return user, nil
}

// Change login password
func (a *Login) UpdatePassword(ctx context.Context, updateItem *sch.UpdateLoginPassword) error {
	if util.FromIsRootUser(ctx) {
		return errors.BadRequest("", "Root user cannot change password")
	}

	userID := util.FromUserID(ctx)
	user, err := a.UserDAL.Get(ctx, userID, sch.UserQueryOptions{
		QueryOptions: util.QueryOptions{
			SelectFields: []string{"password"},
		},
	})
	if err != nil {
		return err
	} else if user == nil {
		return errors.NotFound("", "User not found")
	}

	// check old password
	if err := hash.CompareHashAndPassword(user.Password, updateItem.OldPassword); err != nil {
		return errors.BadRequest("", "Incorrect old password")
	}

	// update password
	newPassword, err := hash.GeneratePassword(updateItem.NewPassword)
	if err != nil {
		return err
	}
	return a.UserDAL.UpdatePasswordByID(ctx, userID, newPassword)
}

// Query menus based on user permissions
func (a *Login) QueryMenus(ctx context.Context) (sch.Menus, error) {
	menuQueryParams := sch.MenuQueryParam{
		Status: sch.MenuStatusEnabled,
	}

	isRoot := util.FromIsRootUser(ctx)
	if !isRoot {
		menuQueryParams.UserID = util.FromUserID(ctx)
	}
	menuResult, err := a.MenuDAL.Query(ctx, menuQueryParams, sch.MenuQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: sch.MenusOrderParams,
		},
	})
	if err != nil {
		return nil, err
	} else if isRoot {
		return menuResult.Data.ToTree(), nil
	}

	// fill parent menus
	if parentIDs := menuResult.Data.SplitParentIDs(); len(parentIDs) > 0 {
		var missMenusIDs []string
		menuIDMapper := menuResult.Data.ToMap()
		for _, parentID := range parentIDs {
			if _, ok := menuIDMapper[parentID]; !ok {
				missMenusIDs = append(missMenusIDs, parentID)
			}
		}
		if len(missMenusIDs) > 0 {
			parentResult, err := a.MenuDAL.Query(ctx, sch.MenuQueryParam{
				InIDs: missMenusIDs,
			})
			if err != nil {
				return nil, err
			}
			menuResult.Data = append(menuResult.Data, parentResult.Data...)
			sort.Sort(menuResult.Data)
		}
	}

	return menuResult.Data.ToTree(), nil
}
