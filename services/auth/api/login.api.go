package api

import (
	"done/services/auth/biz"
	"done/services/auth/sch"

	"done/util"

	"github.com/gin-gonic/gin"
)

type Login struct {
	LoginBIZ *biz.Login
}

// @Tags LoginAPI
// @Summary Login system with username and password
// @Param body body sch.LoginForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=sch.LoginToken}
// @Failure 400 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/login [post]
func (a *Login) Login(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(sch.LoginForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	}

	data, err := a.LoginBIZ.Login(ctx, item.Trim())
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, data)
}

// @Tags LoginAPI
// @Security ApiKeyAuth
// @Summary Logout system
// @Success 200 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/current/logout [post]
func (a *Login) Logout(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.LoginBIZ.Logout(ctx)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags LoginAPI
// @Security ApiKeyAuth
// @Summary Refresh current access token
// @Success 200 {object} util.ResponseResult{data=sch.LoginToken}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/current/refresh-token [post]
func (a *Login) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()
	data, err := a.LoginBIZ.RefreshToken(ctx)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, data)
}

// @Tags LoginAPI
// @Security ApiKeyAuth
// @Summary Get current user info
// @Success 200 {object} util.ResponseResult{data=sch.User}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/current/user [get]
func (a *Login) GetUserInfo(c *gin.Context) {
	ctx := c.Request.Context()
	data, err := a.LoginBIZ.GetUserInfo(ctx)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, data)
}

// @Tags LoginAPI
// @Security ApiKeyAuth
// @Summary Change current user password
// @Param body body sch.UpdateLoginPassword true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/current/password [put]
func (a *Login) UpdatePassword(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(sch.UpdateLoginPassword)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.LoginBIZ.UpdatePassword(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags LoginAPI
// @Security ApiKeyAuth
// @Summary Query current user menus based on the current user role
// @Success 200 {object} util.ResponseResult{data=[]sch.Menu}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/current/menus [get]
func (a *Login) QueryMenus(c *gin.Context) {
	ctx := c.Request.Context()
	data, err := a.LoginBIZ.QueryMenus(ctx)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, data)
}
