package core

import (
	"net/http"

	"done/services/datastorage/daos"
	"done/services/datastorage/models"
	"done/services/datastorage/models/schema"
	"done/tools/filesystem"
	"done/tools/hook"
	"done/tools/search"

	"github.com/ganigeorgiev/echo"
	"golang.org/x/crypto/acme/autocert"
)

var (
	_ hook.Tagger = (*BaseModelEvent)(nil)
	_ hook.Tagger = (*BaseCollectionEvent)(nil)
)

type BaseModelEvent struct {
	Model models.Model
}

func (e *BaseModelEvent) Tags() []string {
	if e.Model == nil {
		return nil
	}

	if r, ok := e.Model.(*models.Record); ok && r.Collection() != nil {
		return []string{r.Collection().Id, r.Collection().Name}
	}

	return []string{e.Model.TableName()}
}

type BaseCollectionEvent struct {
	Collection *models.Collection
}

func (e *BaseCollectionEvent) Tags() []string {
	if e.Collection == nil {
		return nil
	}

	tags := make([]string, 0, 2)

	if e.Collection.Id != "" {
		tags = append(tags, e.Collection.Id)
	}

	if e.Collection.Name != "" {
		tags = append(tags, e.Collection.Name)
	}

	return tags
}

// -------------------------------------------------------------------
// Serve events data
// -------------------------------------------------------------------

type BootstrapEvent struct {
	App App
}

type TerminateEvent struct {
	App App
}

type ServeEvent struct {
	App         App
	Router      *echo.Echo
	Server      *http.Server
	CertManager *autocert.Manager
}

type ApiErrorEvent struct {
	HttpContext echo.Context
	Error       error
}

// -------------------------------------------------------------------
// Model DAO events data
// -------------------------------------------------------------------

type ModelEvent struct {
	BaseModelEvent

	Dao *daos.Dao
}

// -------------------------------------------------------------------
// Record CRUD API events data
// -------------------------------------------------------------------

type RecordsListEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
	Records     []*models.Record
	Result      *search.Result
}

type RecordViewEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
	Record      *models.Record
}

type RecordCreateEvent struct {
	BaseCollectionEvent

	HttpContext   echo.Context
	Record        *models.Record
	UploadedFiles map[string][]*filesystem.File
}

type RecordUpdateEvent struct {
	BaseCollectionEvent

	HttpContext   echo.Context
	Record        *models.Record
	UploadedFiles map[string][]*filesystem.File
}

type RecordDeleteEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
	Record      *models.Record
}

// -------------------------------------------------------------------
// Admin API events data
// -------------------------------------------------------------------

type AdminsListEvent struct {
	HttpContext echo.Context
	Admins      []*models.Admin
	Result      *search.Result
}

type AdminViewEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
}

type AdminCreateEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
}

type AdminUpdateEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
}

type AdminDeleteEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
}

type AdminAuthEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
	Token       string
}

type AdminAuthWithPasswordEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
	Identity    string
	Password    string
}

type AdminAuthRefreshEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
}

type AdminRequestPasswordResetEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
}

type AdminConfirmPasswordResetEvent struct {
	HttpContext echo.Context
	Admin       *models.Admin
}

// -------------------------------------------------------------------
// Collection API events data
// -------------------------------------------------------------------

type CollectionsListEvent struct {
	HttpContext echo.Context
	Collections []*models.Collection
	Result      *search.Result
}

type CollectionViewEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
}

type CollectionCreateEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
}

type CollectionUpdateEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
}

type CollectionDeleteEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
}

type CollectionsImportEvent struct {
	HttpContext echo.Context
	Collections []*models.Collection
}

// -------------------------------------------------------------------
// File API events data
// -------------------------------------------------------------------

type FileTokenEvent struct {
	BaseModelEvent

	HttpContext echo.Context
	Token       string
}

type FileDownloadEvent struct {
	BaseCollectionEvent

	HttpContext echo.Context
	Record      *models.Record
	FileField   *schema.SchemaField
	ServedPath  string
	ServedName  string
}
