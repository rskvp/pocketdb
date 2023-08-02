// Package apis implements the default PocketBase api services and middlewares.
package apis

import (
	"done/services/datastorage/core"

	//"done/services/datastorage/ui"

	"github.com/ganigeorgiev/echo"
)

const trailedAdminPath = "/_/"

// InitApi creates a configured echo instance with registered
// system and app specific routes and middlewares.
func InitApi(app core.App) (*echo.Echo, error) {
	e := echo.New()
	e.Debug = app.IsDebug()

	// default routes
	api := e.Group("/api", eagerRequestInfoCache(app))
	bindAdminApi(app, api)
	bindCollectionApi(app, api)
	bindRecordCrudApi(app, api)

	return e, nil
}
