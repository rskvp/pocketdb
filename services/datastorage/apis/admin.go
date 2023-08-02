package apis

import (
	"done/services/datastorage/core"
	"done/services/datastorage/forms"
	"done/services/datastorage/models"
	"done/services/datastorage/tokens"

	"github.com/ganigeorgiev/echo"
)

// bindAdminApi registers the admin api endpoints and the corresponding handlers.
func bindAdminApi(app core.App, rg *echo.Group) {
	api := adminApi{app: app}

	subGroup := rg.Group("/admins")
	subGroup.POST("/auth-with-password", api.authWithPassword)
	subGroup.POST("/auth-refresh", api.authRefresh, RequireAdminAuth())
}

type adminApi struct {
	app core.App
}

func (api *adminApi) authResponse(c echo.Context, admin *models.Admin, finalizers ...func(token string) error) error {
	token, tokenErr := tokens.NewAdminAuthToken(api.app, admin)
	if tokenErr != nil {
		return NewBadRequestError("Failed to create auth token.", tokenErr)
	}

	for _, f := range finalizers {
		if err := f(token); err != nil {
			return err
		}
	}

	event := new(core.AdminAuthEvent)
	event.HttpContext = c
	event.Admin = admin
	event.Token = token

	return api.app.OnAdminAuthRequest().Trigger(event, func(e *core.AdminAuthEvent) error {
		if e.HttpContext.Response().Committed {
			return nil
		}

		return e.HttpContext.JSON(200, map[string]any{
			"token": e.Token,
			"admin": e.Admin,
		})
	})
}

func (api *adminApi) authRefresh(c echo.Context) error {
	admin, _ := c.Get(ContextAdminKey).(*models.Admin)
	if admin == nil {
		return NewNotFoundError("Missing auth admin context.", nil)
	}

	event := new(core.AdminAuthRefreshEvent)
	event.HttpContext = c
	event.Admin = admin

	return api.app.OnAdminBeforeAuthRefreshRequest().Trigger(event, func(e *core.AdminAuthRefreshEvent) error {
		return api.app.OnAdminAfterAuthRefreshRequest().Trigger(event, func(e *core.AdminAuthRefreshEvent) error {
			return api.authResponse(e.HttpContext, e.Admin)
		})
	})
}

func (api *adminApi) authWithPassword(c echo.Context) error {
	form := forms.NewAdminLogin(api.app)
	if err := c.Bind(form); err != nil {
		return NewBadRequestError("An error occurred while loading the submitted data.", err)
	}

	event := new(core.AdminAuthWithPasswordEvent)
	event.HttpContext = c
	event.Password = form.Password
	event.Identity = form.Identity

	_, submitErr := form.Submit(func(next forms.InterceptorNextFunc[*models.Admin]) forms.InterceptorNextFunc[*models.Admin] {
		return func(admin *models.Admin) error {
			event.Admin = admin

			return api.app.OnAdminBeforeAuthWithPasswordRequest().Trigger(event, func(e *core.AdminAuthWithPasswordEvent) error {
				if err := next(e.Admin); err != nil {
					return NewBadRequestError("Failed to authenticate.", err)
				}

				return api.app.OnAdminAfterAuthWithPasswordRequest().Trigger(event, func(e *core.AdminAuthWithPasswordEvent) error {
					return api.authResponse(e.HttpContext, e.Admin)
				})
			})
		}
	})

	return submitErr
}
