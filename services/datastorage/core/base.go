package core

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"done/services/datastorage/daos"

	"github.com/fatih/color"
	"github.com/pocketbase/dbx"

	//"done/services/datastorage/models/settings"

	"done/tools/hook"
	"done/tools/store"
	"done/tools/subscriptions"
)

const (
	DefaultDataMaxOpenConns int = 120
	DefaultDataMaxIdleConns int = 20
	DefaultLogsMaxOpenConns int = 10
	DefaultLogsMaxIdleConns int = 2

	LocalStorageDirName string = "storage"
	LocalBackupsDirName string = "backups"
	LocalTempDirName    string = ".pb_temp_to_delete" // temp pb_data sub directory that will be deleted on each app.Bootstrap()
)

// BaseApp implements core.App and defines the base PocketBase app structure.
type BaseApp struct {
	// configurable parameters
	isDebug          bool
	dataDir          string
	encryptionEnv    string
	dataMaxOpenConns int
	dataMaxIdleConns int
	logsMaxOpenConns int
	logsMaxIdleConns int

	// internals
	cache               *store.Store[any]
	dao                 *daos.Dao
	logsDao             *daos.Dao
	subscriptionsBroker *subscriptions.Broker

	// app event hooks
	onBeforeBootstrap *hook.Hook[*BootstrapEvent]
	onAfterBootstrap  *hook.Hook[*BootstrapEvent]
	onBeforeServe     *hook.Hook[*ServeEvent]
	onBeforeApiError  *hook.Hook[*ApiErrorEvent]
	onAfterApiError   *hook.Hook[*ApiErrorEvent]
	onTerminate       *hook.Hook[*TerminateEvent]

	// dao event hooks
	onModelBeforeCreate *hook.Hook[*ModelEvent]
	onModelAfterCreate  *hook.Hook[*ModelEvent]
	onModelBeforeUpdate *hook.Hook[*ModelEvent]
	onModelAfterUpdate  *hook.Hook[*ModelEvent]
	onModelBeforeDelete *hook.Hook[*ModelEvent]
	onModelAfterDelete  *hook.Hook[*ModelEvent]

	// file api event hooks
	onFileDownloadRequest    *hook.Hook[*FileDownloadEvent]
	onFileBeforeTokenRequest *hook.Hook[*FileTokenEvent]
	onFileAfterTokenRequest  *hook.Hook[*FileTokenEvent]

	// admin api event hooks
	onAdminsListRequest                      *hook.Hook[*AdminsListEvent]
	onAdminViewRequest                       *hook.Hook[*AdminViewEvent]
	onAdminBeforeCreateRequest               *hook.Hook[*AdminCreateEvent]
	onAdminAfterCreateRequest                *hook.Hook[*AdminCreateEvent]
	onAdminBeforeUpdateRequest               *hook.Hook[*AdminUpdateEvent]
	onAdminAfterUpdateRequest                *hook.Hook[*AdminUpdateEvent]
	onAdminBeforeDeleteRequest               *hook.Hook[*AdminDeleteEvent]
	onAdminAfterDeleteRequest                *hook.Hook[*AdminDeleteEvent]
	onAdminAuthRequest                       *hook.Hook[*AdminAuthEvent]
	onAdminBeforeAuthWithPasswordRequest     *hook.Hook[*AdminAuthWithPasswordEvent]
	onAdminAfterAuthWithPasswordRequest      *hook.Hook[*AdminAuthWithPasswordEvent]
	onAdminBeforeAuthRefreshRequest          *hook.Hook[*AdminAuthRefreshEvent]
	onAdminAfterAuthRefreshRequest           *hook.Hook[*AdminAuthRefreshEvent]
	onAdminBeforeRequestPasswordResetRequest *hook.Hook[*AdminRequestPasswordResetEvent]
	onAdminAfterRequestPasswordResetRequest  *hook.Hook[*AdminRequestPasswordResetEvent]
	onAdminBeforeConfirmPasswordResetRequest *hook.Hook[*AdminConfirmPasswordResetEvent]
	onAdminAfterConfirmPasswordResetRequest  *hook.Hook[*AdminConfirmPasswordResetEvent]

	// record crud API event hooks
	onRecordsListRequest        *hook.Hook[*RecordsListEvent]
	onRecordViewRequest         *hook.Hook[*RecordViewEvent]
	onRecordBeforeCreateRequest *hook.Hook[*RecordCreateEvent]
	onRecordAfterCreateRequest  *hook.Hook[*RecordCreateEvent]
	onRecordBeforeUpdateRequest *hook.Hook[*RecordUpdateEvent]
	onRecordAfterUpdateRequest  *hook.Hook[*RecordUpdateEvent]
	onRecordBeforeDeleteRequest *hook.Hook[*RecordDeleteEvent]
	onRecordAfterDeleteRequest  *hook.Hook[*RecordDeleteEvent]

	// collection API event hooks
	onCollectionsListRequest         *hook.Hook[*CollectionsListEvent]
	onCollectionViewRequest          *hook.Hook[*CollectionViewEvent]
	onCollectionBeforeCreateRequest  *hook.Hook[*CollectionCreateEvent]
	onCollectionAfterCreateRequest   *hook.Hook[*CollectionCreateEvent]
	onCollectionBeforeUpdateRequest  *hook.Hook[*CollectionUpdateEvent]
	onCollectionAfterUpdateRequest   *hook.Hook[*CollectionUpdateEvent]
	onCollectionBeforeDeleteRequest  *hook.Hook[*CollectionDeleteEvent]
	onCollectionAfterDeleteRequest   *hook.Hook[*CollectionDeleteEvent]
	onCollectionsBeforeImportRequest *hook.Hook[*CollectionsImportEvent]
	onCollectionsAfterImportRequest  *hook.Hook[*CollectionsImportEvent]
}

// BaseAppConfig defines a BaseApp configuration option
type BaseAppConfig struct {
	DataDir          string
	EncryptionEnv    string
	IsDebug          bool
	DataMaxOpenConns int // default to 500
	DataMaxIdleConns int // default 20
	LogsMaxOpenConns int // default to 100
	LogsMaxIdleConns int // default to 5
}

// NewBaseApp creates and returns a new BaseApp instance
// configured with the provided arguments.
//
// To initialize the app, you need to call `app.Bootstrap()`.
func NewBaseApp(config BaseAppConfig) *BaseApp {
	app := &BaseApp{
		dataDir:             config.DataDir,
		isDebug:             config.IsDebug,
		encryptionEnv:       config.EncryptionEnv,
		dataMaxOpenConns:    config.DataMaxOpenConns,
		dataMaxIdleConns:    config.DataMaxIdleConns,
		logsMaxOpenConns:    config.LogsMaxOpenConns,
		logsMaxIdleConns:    config.LogsMaxIdleConns,
		cache:               store.New[any](nil),
		subscriptionsBroker: subscriptions.NewBroker(),

		// app event hooks
		onBeforeBootstrap: &hook.Hook[*BootstrapEvent]{},
		onAfterBootstrap:  &hook.Hook[*BootstrapEvent]{},
		onBeforeServe:     &hook.Hook[*ServeEvent]{},
		onBeforeApiError:  &hook.Hook[*ApiErrorEvent]{},
		onAfterApiError:   &hook.Hook[*ApiErrorEvent]{},
		onTerminate:       &hook.Hook[*TerminateEvent]{},

		// dao event hooks
		onModelBeforeCreate: &hook.Hook[*ModelEvent]{},
		onModelAfterCreate:  &hook.Hook[*ModelEvent]{},
		onModelBeforeUpdate: &hook.Hook[*ModelEvent]{},
		onModelAfterUpdate:  &hook.Hook[*ModelEvent]{},
		onModelBeforeDelete: &hook.Hook[*ModelEvent]{},
		onModelAfterDelete:  &hook.Hook[*ModelEvent]{},

		// file API event hooks
		onFileDownloadRequest:    &hook.Hook[*FileDownloadEvent]{},
		onFileBeforeTokenRequest: &hook.Hook[*FileTokenEvent]{},
		onFileAfterTokenRequest:  &hook.Hook[*FileTokenEvent]{},

		// admin API event hooks
		onAdminsListRequest:                      &hook.Hook[*AdminsListEvent]{},
		onAdminViewRequest:                       &hook.Hook[*AdminViewEvent]{},
		onAdminBeforeCreateRequest:               &hook.Hook[*AdminCreateEvent]{},
		onAdminAfterCreateRequest:                &hook.Hook[*AdminCreateEvent]{},
		onAdminBeforeUpdateRequest:               &hook.Hook[*AdminUpdateEvent]{},
		onAdminAfterUpdateRequest:                &hook.Hook[*AdminUpdateEvent]{},
		onAdminBeforeDeleteRequest:               &hook.Hook[*AdminDeleteEvent]{},
		onAdminAfterDeleteRequest:                &hook.Hook[*AdminDeleteEvent]{},
		onAdminAuthRequest:                       &hook.Hook[*AdminAuthEvent]{},
		onAdminBeforeAuthWithPasswordRequest:     &hook.Hook[*AdminAuthWithPasswordEvent]{},
		onAdminAfterAuthWithPasswordRequest:      &hook.Hook[*AdminAuthWithPasswordEvent]{},
		onAdminBeforeAuthRefreshRequest:          &hook.Hook[*AdminAuthRefreshEvent]{},
		onAdminAfterAuthRefreshRequest:           &hook.Hook[*AdminAuthRefreshEvent]{},
		onAdminBeforeRequestPasswordResetRequest: &hook.Hook[*AdminRequestPasswordResetEvent]{},
		onAdminAfterRequestPasswordResetRequest:  &hook.Hook[*AdminRequestPasswordResetEvent]{},
		onAdminBeforeConfirmPasswordResetRequest: &hook.Hook[*AdminConfirmPasswordResetEvent]{},
		onAdminAfterConfirmPasswordResetRequest:  &hook.Hook[*AdminConfirmPasswordResetEvent]{},

		// record crud API event hooks
		onRecordsListRequest:        &hook.Hook[*RecordsListEvent]{},
		onRecordViewRequest:         &hook.Hook[*RecordViewEvent]{},
		onRecordBeforeCreateRequest: &hook.Hook[*RecordCreateEvent]{},
		onRecordAfterCreateRequest:  &hook.Hook[*RecordCreateEvent]{},
		onRecordBeforeUpdateRequest: &hook.Hook[*RecordUpdateEvent]{},
		onRecordAfterUpdateRequest:  &hook.Hook[*RecordUpdateEvent]{},
		onRecordBeforeDeleteRequest: &hook.Hook[*RecordDeleteEvent]{},
		onRecordAfterDeleteRequest:  &hook.Hook[*RecordDeleteEvent]{},

		// collection API event hooks
		onCollectionsListRequest:         &hook.Hook[*CollectionsListEvent]{},
		onCollectionViewRequest:          &hook.Hook[*CollectionViewEvent]{},
		onCollectionBeforeCreateRequest:  &hook.Hook[*CollectionCreateEvent]{},
		onCollectionAfterCreateRequest:   &hook.Hook[*CollectionCreateEvent]{},
		onCollectionBeforeUpdateRequest:  &hook.Hook[*CollectionUpdateEvent]{},
		onCollectionAfterUpdateRequest:   &hook.Hook[*CollectionUpdateEvent]{},
		onCollectionBeforeDeleteRequest:  &hook.Hook[*CollectionDeleteEvent]{},
		onCollectionAfterDeleteRequest:   &hook.Hook[*CollectionDeleteEvent]{},
		onCollectionsBeforeImportRequest: &hook.Hook[*CollectionsImportEvent]{},
		onCollectionsAfterImportRequest:  &hook.Hook[*CollectionsImportEvent]{},
	}

	return app
}

// Bootstrap initializes the application
// (aka. create data dir, open db connections, load settings, etc.).
//
// It will call ResetBootstrapState() if the application was already bootstrapped.
func (app *BaseApp) Bootstrap() error {

	// clear resources of previous core state (if any)
	if err := app.ResetBootstrapState(); err != nil {
		return err
	}

	// ensure that data dir exist
	if err := os.MkdirAll(app.DataDir(), os.ModePerm); err != nil {
		return err
	}

	if err := app.initDataDB(); err != nil {
		return err
	}

	if err := app.initLogsDB(); err != nil {
		return err
	}

	// we don't check for an error because the db migrations may have not been executed yet

	// cleanup the pb_data temp directory (if any)
	os.RemoveAll(filepath.Join(app.DataDir(), LocalTempDirName))
	return nil
}

// ResetBootstrapState takes care for releasing initialized app resources
// (eg. closing db connections).
func (app *BaseApp) ResetBootstrapState() error {
	if app.Dao() != nil {
		if err := app.Dao().ConcurrentDB().(*dbx.DB).Close(); err != nil {
			return err
		}
		if err := app.Dao().NonconcurrentDB().(*dbx.DB).Close(); err != nil {
			return err
		}
	}

	if app.LogsDao() != nil {
		if err := app.LogsDao().ConcurrentDB().(*dbx.DB).Close(); err != nil {
			return err
		}
		if err := app.LogsDao().NonconcurrentDB().(*dbx.DB).Close(); err != nil {
			return err
		}
	}

	app.dao = nil
	app.logsDao = nil

	return nil
}

// Deprecated:
// This method may get removed in the near future.
// It is recommended to access the db instance from app.Dao().DB() or
// if you want more flexibility - app.Dao().ConcurrentDB() and app.Dao().NonconcurrentDB().
//
// DB returns the default app database instance.
func (app *BaseApp) DB() *dbx.DB {
	if app.Dao() == nil {
		return nil
	}

	db, ok := app.Dao().DB().(*dbx.DB)
	if !ok {
		return nil
	}

	return db
}

// Dao returns the default app Dao instance.
func (app *BaseApp) Dao() *daos.Dao {
	return app.dao
}

// Deprecated:
// This method may get removed in the near future.
// It is recommended to access the logs db instance from app.LogsDao().DB() or
// if you want more flexibility - app.LogsDao().ConcurrentDB() and app.LogsDao().NonconcurrentDB().
//
// LogsDB returns the app logs database instance.
func (app *BaseApp) LogsDB() *dbx.DB {
	if app.LogsDao() == nil {
		return nil
	}

	db, ok := app.LogsDao().DB().(*dbx.DB)
	if !ok {
		return nil
	}

	return db
}

// LogsDao returns the app logs Dao instance.
func (app *BaseApp) LogsDao() *daos.Dao {
	return app.logsDao
}

// DataDir returns the app data directory path.
func (app *BaseApp) DataDir() string {
	return app.dataDir
}

// EncryptionEnv returns the name of the app secret env key
// (used for settings encryption).
func (app *BaseApp) EncryptionEnv() string {
	return app.encryptionEnv
}

// IsDebug returns whether the app is in debug mode
// (showing more detailed error logs, executed sql statements, etc.).
func (app *BaseApp) IsDebug() bool {
	return app.isDebug
}

// Cache returns the app internal cache store.
func (app *BaseApp) Cache() *store.Store[any] {
	return app.cache
}

// SubscriptionsBroker returns the app realtime subscriptions broker instance.
func (app *BaseApp) SubscriptionsBroker() *subscriptions.Broker {
	return app.subscriptionsBroker
}

// Restart restarts (aka. replaces) the current running application process.
//
// NB! It relies on execve which is supported only on UNIX based systems.
func (app *BaseApp) Restart() error {
	if runtime.GOOS == "windows" {
		return errors.New("restart is not supported on windows")
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// optimistically reset the app bootstrap state
	app.ResetBootstrapState()

	if err := syscall.Exec(execPath, os.Args, os.Environ()); err != nil {
		// restart the app bootstrap state
		app.Bootstrap()

		return err
	}

	return nil
}

// -------------------------------------------------------------------
// App event hooks
// -------------------------------------------------------------------

func (app *BaseApp) OnBeforeBootstrap() *hook.Hook[*BootstrapEvent] {
	return app.onBeforeBootstrap
}

func (app *BaseApp) OnAfterBootstrap() *hook.Hook[*BootstrapEvent] {
	return app.onAfterBootstrap
}

func (app *BaseApp) OnBeforeServe() *hook.Hook[*ServeEvent] {
	return app.onBeforeServe
}

func (app *BaseApp) OnBeforeApiError() *hook.Hook[*ApiErrorEvent] {
	return app.onBeforeApiError
}

func (app *BaseApp) OnAfterApiError() *hook.Hook[*ApiErrorEvent] {
	return app.onAfterApiError
}

func (app *BaseApp) OnTerminate() *hook.Hook[*TerminateEvent] {
	return app.onTerminate
}

// -------------------------------------------------------------------
// Dao event hooks
// -------------------------------------------------------------------

func (app *BaseApp) OnModelBeforeCreate(tags ...string) *hook.TaggedHook[*ModelEvent] {
	return hook.NewTaggedHook(app.onModelBeforeCreate, tags...)
}

func (app *BaseApp) OnModelAfterCreate(tags ...string) *hook.TaggedHook[*ModelEvent] {
	return hook.NewTaggedHook(app.onModelAfterCreate, tags...)
}

func (app *BaseApp) OnModelBeforeUpdate(tags ...string) *hook.TaggedHook[*ModelEvent] {
	return hook.NewTaggedHook(app.onModelBeforeUpdate, tags...)
}

func (app *BaseApp) OnModelAfterUpdate(tags ...string) *hook.TaggedHook[*ModelEvent] {
	return hook.NewTaggedHook(app.onModelAfterUpdate, tags...)
}

func (app *BaseApp) OnModelBeforeDelete(tags ...string) *hook.TaggedHook[*ModelEvent] {
	return hook.NewTaggedHook(app.onModelBeforeDelete, tags...)
}

func (app *BaseApp) OnModelAfterDelete(tags ...string) *hook.TaggedHook[*ModelEvent] {
	return hook.NewTaggedHook(app.onModelAfterDelete, tags...)
}

// -------------------------------------------------------------------
// File API event hooks
// -------------------------------------------------------------------

func (app *BaseApp) OnFileDownloadRequest(tags ...string) *hook.TaggedHook[*FileDownloadEvent] {
	return hook.NewTaggedHook(app.onFileDownloadRequest, tags...)
}

func (app *BaseApp) OnFileBeforeTokenRequest(tags ...string) *hook.TaggedHook[*FileTokenEvent] {
	return hook.NewTaggedHook(app.onFileBeforeTokenRequest, tags...)
}

func (app *BaseApp) OnFileAfterTokenRequest(tags ...string) *hook.TaggedHook[*FileTokenEvent] {
	return hook.NewTaggedHook(app.onFileAfterTokenRequest, tags...)
}

// -------------------------------------------------------------------
// Admin API event hooks
// -------------------------------------------------------------------

func (app *BaseApp) OnAdminsListRequest() *hook.Hook[*AdminsListEvent] {
	return app.onAdminsListRequest
}

func (app *BaseApp) OnAdminViewRequest() *hook.Hook[*AdminViewEvent] {
	return app.onAdminViewRequest
}

func (app *BaseApp) OnAdminBeforeCreateRequest() *hook.Hook[*AdminCreateEvent] {
	return app.onAdminBeforeCreateRequest
}

func (app *BaseApp) OnAdminAfterCreateRequest() *hook.Hook[*AdminCreateEvent] {
	return app.onAdminAfterCreateRequest
}

func (app *BaseApp) OnAdminBeforeUpdateRequest() *hook.Hook[*AdminUpdateEvent] {
	return app.onAdminBeforeUpdateRequest
}

func (app *BaseApp) OnAdminAfterUpdateRequest() *hook.Hook[*AdminUpdateEvent] {
	return app.onAdminAfterUpdateRequest
}

func (app *BaseApp) OnAdminBeforeDeleteRequest() *hook.Hook[*AdminDeleteEvent] {
	return app.onAdminBeforeDeleteRequest
}

func (app *BaseApp) OnAdminAfterDeleteRequest() *hook.Hook[*AdminDeleteEvent] {
	return app.onAdminAfterDeleteRequest
}

func (app *BaseApp) OnAdminAuthRequest() *hook.Hook[*AdminAuthEvent] {
	return app.onAdminAuthRequest
}

func (app *BaseApp) OnAdminBeforeAuthWithPasswordRequest() *hook.Hook[*AdminAuthWithPasswordEvent] {
	return app.onAdminBeforeAuthWithPasswordRequest
}

func (app *BaseApp) OnAdminAfterAuthWithPasswordRequest() *hook.Hook[*AdminAuthWithPasswordEvent] {
	return app.onAdminAfterAuthWithPasswordRequest
}

func (app *BaseApp) OnAdminBeforeAuthRefreshRequest() *hook.Hook[*AdminAuthRefreshEvent] {
	return app.onAdminBeforeAuthRefreshRequest
}

func (app *BaseApp) OnAdminAfterAuthRefreshRequest() *hook.Hook[*AdminAuthRefreshEvent] {
	return app.onAdminAfterAuthRefreshRequest
}

func (app *BaseApp) OnAdminBeforeRequestPasswordResetRequest() *hook.Hook[*AdminRequestPasswordResetEvent] {
	return app.onAdminBeforeRequestPasswordResetRequest
}

func (app *BaseApp) OnAdminAfterRequestPasswordResetRequest() *hook.Hook[*AdminRequestPasswordResetEvent] {
	return app.onAdminAfterRequestPasswordResetRequest
}

func (app *BaseApp) OnAdminBeforeConfirmPasswordResetRequest() *hook.Hook[*AdminConfirmPasswordResetEvent] {
	return app.onAdminBeforeConfirmPasswordResetRequest
}

func (app *BaseApp) OnAdminAfterConfirmPasswordResetRequest() *hook.Hook[*AdminConfirmPasswordResetEvent] {
	return app.onAdminAfterConfirmPasswordResetRequest
}

// -------------------------------------------------------------------
// Record CRUD API event hooks
// -------------------------------------------------------------------

func (app *BaseApp) OnRecordsListRequest(tags ...string) *hook.TaggedHook[*RecordsListEvent] {
	return hook.NewTaggedHook(app.onRecordsListRequest, tags...)
}

func (app *BaseApp) OnRecordViewRequest(tags ...string) *hook.TaggedHook[*RecordViewEvent] {
	return hook.NewTaggedHook(app.onRecordViewRequest, tags...)
}

func (app *BaseApp) OnRecordBeforeCreateRequest(tags ...string) *hook.TaggedHook[*RecordCreateEvent] {
	return hook.NewTaggedHook(app.onRecordBeforeCreateRequest, tags...)
}

func (app *BaseApp) OnRecordAfterCreateRequest(tags ...string) *hook.TaggedHook[*RecordCreateEvent] {
	return hook.NewTaggedHook(app.onRecordAfterCreateRequest, tags...)
}

func (app *BaseApp) OnRecordBeforeUpdateRequest(tags ...string) *hook.TaggedHook[*RecordUpdateEvent] {
	return hook.NewTaggedHook(app.onRecordBeforeUpdateRequest, tags...)
}

func (app *BaseApp) OnRecordAfterUpdateRequest(tags ...string) *hook.TaggedHook[*RecordUpdateEvent] {
	return hook.NewTaggedHook(app.onRecordAfterUpdateRequest, tags...)
}

func (app *BaseApp) OnRecordBeforeDeleteRequest(tags ...string) *hook.TaggedHook[*RecordDeleteEvent] {
	return hook.NewTaggedHook(app.onRecordBeforeDeleteRequest, tags...)
}

func (app *BaseApp) OnRecordAfterDeleteRequest(tags ...string) *hook.TaggedHook[*RecordDeleteEvent] {
	return hook.NewTaggedHook(app.onRecordAfterDeleteRequest, tags...)
}

// -------------------------------------------------------------------
// Collection API event hooks
// -------------------------------------------------------------------

func (app *BaseApp) OnCollectionsListRequest() *hook.Hook[*CollectionsListEvent] {
	return app.onCollectionsListRequest
}

func (app *BaseApp) OnCollectionViewRequest() *hook.Hook[*CollectionViewEvent] {
	return app.onCollectionViewRequest
}

func (app *BaseApp) OnCollectionBeforeCreateRequest() *hook.Hook[*CollectionCreateEvent] {
	return app.onCollectionBeforeCreateRequest
}

func (app *BaseApp) OnCollectionAfterCreateRequest() *hook.Hook[*CollectionCreateEvent] {
	return app.onCollectionAfterCreateRequest
}

func (app *BaseApp) OnCollectionBeforeUpdateRequest() *hook.Hook[*CollectionUpdateEvent] {
	return app.onCollectionBeforeUpdateRequest
}

func (app *BaseApp) OnCollectionAfterUpdateRequest() *hook.Hook[*CollectionUpdateEvent] {
	return app.onCollectionAfterUpdateRequest
}

func (app *BaseApp) OnCollectionBeforeDeleteRequest() *hook.Hook[*CollectionDeleteEvent] {
	return app.onCollectionBeforeDeleteRequest
}

func (app *BaseApp) OnCollectionAfterDeleteRequest() *hook.Hook[*CollectionDeleteEvent] {
	return app.onCollectionAfterDeleteRequest
}

func (app *BaseApp) OnCollectionsBeforeImportRequest() *hook.Hook[*CollectionsImportEvent] {
	return app.onCollectionsBeforeImportRequest
}

func (app *BaseApp) OnCollectionsAfterImportRequest() *hook.Hook[*CollectionsImportEvent] {
	return app.onCollectionsAfterImportRequest
}

// -------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------

func (app *BaseApp) initLogsDB() error {
	maxOpenConns := DefaultLogsMaxOpenConns
	maxIdleConns := DefaultLogsMaxIdleConns
	if app.logsMaxOpenConns > 0 {
		maxOpenConns = app.logsMaxOpenConns
	}
	if app.logsMaxIdleConns > 0 {
		maxIdleConns = app.logsMaxIdleConns
	}

	concurrentDB, err := connectDB(filepath.Join(app.DataDir(), "logs.db"))
	if err != nil {
		return err
	}
	concurrentDB.DB().SetMaxOpenConns(maxOpenConns)
	concurrentDB.DB().SetMaxIdleConns(maxIdleConns)
	concurrentDB.DB().SetConnMaxIdleTime(5 * time.Minute)

	nonconcurrentDB, err := connectDB(filepath.Join(app.DataDir(), "logs.db"))
	if err != nil {
		return err
	}
	nonconcurrentDB.DB().SetMaxOpenConns(1)
	nonconcurrentDB.DB().SetMaxIdleConns(1)
	nonconcurrentDB.DB().SetConnMaxIdleTime(5 * time.Minute)

	app.logsDao = daos.NewMultiDB(concurrentDB, nonconcurrentDB)

	return nil
}

func (app *BaseApp) initDataDB() error {
	maxOpenConns := DefaultDataMaxOpenConns
	maxIdleConns := DefaultDataMaxIdleConns
	if app.dataMaxOpenConns > 0 {
		maxOpenConns = app.dataMaxOpenConns
	}
	if app.dataMaxIdleConns > 0 {
		maxIdleConns = app.dataMaxIdleConns
	}

	concurrentDB, err := connectDB(filepath.Join(app.DataDir(), "data.db"))
	if err != nil {
		return err
	}
	concurrentDB.DB().SetMaxOpenConns(maxOpenConns)
	concurrentDB.DB().SetMaxIdleConns(maxIdleConns)
	concurrentDB.DB().SetConnMaxIdleTime(5 * time.Minute)

	nonconcurrentDB, err := connectDB(filepath.Join(app.DataDir(), "data.db"))
	if err != nil {
		return err
	}
	nonconcurrentDB.DB().SetMaxOpenConns(1)
	nonconcurrentDB.DB().SetMaxIdleConns(1)
	nonconcurrentDB.DB().SetConnMaxIdleTime(5 * time.Minute)

	if app.IsDebug() {
		nonconcurrentDB.QueryLogFunc = func(ctx context.Context, t time.Duration, sql string, rows *sql.Rows, err error) {
			color.HiBlack("[%.2fms] %v\n", float64(t.Milliseconds()), sql)
		}
		concurrentDB.QueryLogFunc = nonconcurrentDB.QueryLogFunc

		nonconcurrentDB.ExecLogFunc = func(ctx context.Context, t time.Duration, sql string, result sql.Result, err error) {
			color.HiBlack("[%.2fms] %v\n", float64(t.Milliseconds()), sql)
		}
		concurrentDB.ExecLogFunc = nonconcurrentDB.ExecLogFunc
	}

	return nil
}
