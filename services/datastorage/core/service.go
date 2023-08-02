// Package core is the backbone of PocketBase.
//
// It defines the main PocketBase App interface and its base implementation.
package core

import (
	"context"

	"done/services/datastorage/daos"
	"done/tools/filesystem"
	"done/tools/hook"
	"done/tools/mailer"
	"done/tools/store"
	"done/tools/subscriptions"

	"github.com/pocketbase/dbx"
)

// App defines the main PocketBase app interface.
type App interface {
	// Deprecated:
	// This method may get removed in the near future.
	// It is recommended to access the app db instance from app.Dao().DB() or
	// if you want more flexibility - app.Dao().ConcurrentDB() and app.Dao().NonconcurrentDB().
	//
	// DB returns the default app database instance.
	DB() *dbx.DB

	// Dao returns the default app Dao instance.
	//
	// This Dao could operate only on the tables and models
	// associated with the default app database. For example,
	// trying to access the request logs table will result in error.
	Dao() *daos.Dao

	// Deprecated:
	// This method may get removed in the near future.
	// It is recommended to access the logs db instance from app.LogsDao().DB() or
	// if you want more flexibility - app.LogsDao().ConcurrentDB() and app.LogsDao().NonconcurrentDB().
	//
	// LogsDB returns the app logs database instance.
	LogsDB() *dbx.DB

	// LogsDao returns the app logs Dao instance.
	//
	// This Dao could operate only on the tables and models
	// associated with the logs database. For example, trying to access
	// the users table from LogsDao will result in error.
	LogsDao() *daos.Dao

	// DataDir returns the app data directory path.
	DataDir() string

	// EncryptionEnv returns the name of the app secret env key
	// (used for settings encryption).
	EncryptionEnv() string

	// IsDebug returns whether the app is in debug mode
	// (showing more detailed error logs, executed sql statements, etc.).
	IsDebug() bool

	// Cache returns the app internal cache store.
	Cache() *store.Store[any]

	// SubscriptionsBroker returns the app realtime subscriptions broker instance.
	SubscriptionsBroker() *subscriptions.Broker

	// NewMailClient creates and returns a configured app mail client.
	NewMailClient() mailer.Mailer

	// NewFilesystem creates and returns a configured filesystem.System instance
	// for managing regular app files (eg. collection uploads).
	//
	// NB! Make sure to call Close() on the returned result
	// after you are done working with it.
	NewFilesystem() (*filesystem.System, error)

	// NewBackupsFilesystem creates and returns a configured filesystem.System instance
	// for managing app backups.
	//
	// NB! Make sure to call Close() on the returned result
	// after you are done working with it.
	NewBackupsFilesystem() (*filesystem.System, error)

	// RefreshSettings reinitializes and reloads the stored application settings.
	RefreshSettings() error

	// IsBootstrapped checks if the application was initialized
	// (aka. whether Bootstrap() was called).
	IsBootstrapped() bool

	// Bootstrap takes care for initializing the application
	// (open db connections, load settings, etc.).
	//
	// It will call ResetBootstrapState() if the application was already bootstrapped.
	Bootstrap() error

	// ResetBootstrapState takes care for releasing initialized app resources
	// (eg. closing db connections).
	ResetBootstrapState() error

	// CreateBackup creates a new backup of the current app pb_data directory.
	//
	// Backups can be stored on S3 if it is configured in app.Settings().Backups.
	//
	// Please refer to the godoc of the specific core.App implementation
	// for details on the backup procedures.
	CreateBackup(ctx context.Context, name string) error

	// RestoreBackup restores the backup with the specified name and restarts
	// the current running application process.
	//
	// The safely perform the restore it is recommended to have free disk space
	// for at least 2x the size of the restored pb_data backup.
	//
	// Please refer to the godoc of the specific core.App implementation
	// for details on the restore procedures.
	//
	// NB! This feature is experimental and currently is expected to work only on UNIX based systems.
	RestoreBackup(ctx context.Context, name string) error

	// Restart restarts the current running application process.
	//
	// Currently it is relying on execve so it is supported only on UNIX based systems.
	Restart() error

	// ---------------------------------------------------------------
	// App event hooks
	// ---------------------------------------------------------------

	// OnBeforeBootstrap hook is triggered before initializing the main
	// application resources (eg. before db open and initial settings load).
	OnBeforeBootstrap() *hook.Hook[*BootstrapEvent]

	// OnAfterBootstrap hook is triggered after initializing the main
	// application resources (eg. after db open and initial settings load).
	OnAfterBootstrap() *hook.Hook[*BootstrapEvent]

	// OnBeforeServe hook is triggered before serving the internal router (echo),
	// allowing you to adjust its options and attach new routes or middlewares.
	OnBeforeServe() *hook.Hook[*ServeEvent]

	// OnBeforeApiError hook is triggered right before sending an error API
	// response to the client, allowing you to further modify the error data
	// or to return a completely different API response.
	OnBeforeApiError() *hook.Hook[*ApiErrorEvent]

	// OnAfterApiError hook is triggered right after sending an error API
	// response to the client.
	// It could be used to log the final API error in external services.
	OnAfterApiError() *hook.Hook[*ApiErrorEvent]

	// OnTerminate hook is triggered when the app is in the process
	// of being terminated (eg. on SIGTERM signal).
	OnTerminate() *hook.Hook[*TerminateEvent]

	// ---------------------------------------------------------------
	// Dao event hooks
	// ---------------------------------------------------------------

	// OnModelBeforeCreate hook is triggered before inserting a new
	// model in the DB, allowing you to modify or validate the stored data.
	//
	// If the optional "tags" list (table names and/or the Collection id for Record models)
	// is specified, then all event handlers registered via the created hook
	// will be triggered and called only if their event data origin matches the tags.
	OnModelBeforeCreate(tags ...string) *hook.TaggedHook[*ModelEvent]

	// OnModelAfterCreate hook is triggered after successfully
	// inserting a new model in the DB.
	//
	// If the optional "tags" list (table names and/or the Collection id for Record models)
	// is specified, then all event handlers registered via the created hook
	// will be triggered and called only if their event data origin matches the tags.
	OnModelAfterCreate(tags ...string) *hook.TaggedHook[*ModelEvent]

	// OnModelBeforeUpdate hook is triggered before updating existing
	// model in the DB, allowing you to modify or validate the stored data.
	//
	// If the optional "tags" list (table names and/or the Collection id for Record models)
	// is specified, then all event handlers registered via the created hook
	// will be triggered and called only if their event data origin matches the tags.
	OnModelBeforeUpdate(tags ...string) *hook.TaggedHook[*ModelEvent]

	// OnModelAfterUpdate hook is triggered after successfully updating
	// existing model in the DB.
	//
	// If the optional "tags" list (table names and/or the Collection id for Record models)
	// is specified, then all event handlers registered via the created hook
	// will be triggered and called only if their event data origin matches the tags.
	OnModelAfterUpdate(tags ...string) *hook.TaggedHook[*ModelEvent]

	// OnModelBeforeDelete hook is triggered before deleting an
	// existing model from the DB.
	//
	// If the optional "tags" list (table names and/or the Collection id for Record models)
	// is specified, then all event handlers registered via the created hook
	// will be triggered and called only if their event data origin matches the tags.
	OnModelBeforeDelete(tags ...string) *hook.TaggedHook[*ModelEvent]

	// OnModelAfterDelete hook is triggered after successfully deleting an
	// existing model from the DB.
	//
	// If the optional "tags" list (table names and/or the Collection id for Record models)
	// is specified, then all event handlers registered via the created hook
	// will be triggered and called only if their event data origin matches the tags.
	OnModelAfterDelete(tags ...string) *hook.TaggedHook[*ModelEvent]

	// ---------------------------------------------------------------
	// File API event hooks
	// ---------------------------------------------------------------

	// OnFileDownloadRequest hook is triggered before each API File download request.
	//
	// Could be used to validate or modify the file response before
	// returning it to the client.
	OnFileDownloadRequest(tags ...string) *hook.TaggedHook[*FileDownloadEvent]

	// OnFileBeforeTokenRequest hook is triggered before each file
	// token API request.
	//
	// If no token or model was submitted, e.Model and e.Token will be empty,
	// allowing you to implement your own custom model file auth implementation.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnFileBeforeTokenRequest(tags ...string) *hook.TaggedHook[*FileTokenEvent]

	// OnFileAfterTokenRequest hook is triggered after each
	// successful file token API request.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnFileAfterTokenRequest(tags ...string) *hook.TaggedHook[*FileTokenEvent]

	// ---------------------------------------------------------------
	// Admin API event hooks
	// ---------------------------------------------------------------

	// OnAdminsListRequest hook is triggered on each API Admins list request.
	//
	// Could be used to validate or modify the response before returning it to the client.
	OnAdminsListRequest() *hook.Hook[*AdminsListEvent]

	// OnAdminViewRequest hook is triggered on each API Admin view request.
	//
	// Could be used to validate or modify the response before returning it to the client.
	OnAdminViewRequest() *hook.Hook[*AdminViewEvent]

	// OnAdminBeforeCreateRequest hook is triggered before each API
	// Admin create request (after request data load and before model persistence).
	//
	// Could be used to additionally validate the request data or implement
	// completely different persistence behavior.
	OnAdminBeforeCreateRequest() *hook.Hook[*AdminCreateEvent]

	// OnAdminAfterCreateRequest hook is triggered after each
	// successful API Admin create request.
	OnAdminAfterCreateRequest() *hook.Hook[*AdminCreateEvent]

	// OnAdminBeforeUpdateRequest hook is triggered before each API
	// Admin update request (after request data load and before model persistence).
	//
	// Could be used to additionally validate the request data or implement
	// completely different persistence behavior.
	OnAdminBeforeUpdateRequest() *hook.Hook[*AdminUpdateEvent]

	// OnAdminAfterUpdateRequest hook is triggered after each
	// successful API Admin update request.
	OnAdminAfterUpdateRequest() *hook.Hook[*AdminUpdateEvent]

	// OnAdminBeforeDeleteRequest hook is triggered before each API
	// Admin delete request (after model load and before actual deletion).
	//
	// Could be used to additionally validate the request data or implement
	// completely different delete behavior.
	OnAdminBeforeDeleteRequest() *hook.Hook[*AdminDeleteEvent]

	// OnAdminAfterDeleteRequest hook is triggered after each
	// successful API Admin delete request.
	OnAdminAfterDeleteRequest() *hook.Hook[*AdminDeleteEvent]

	// OnAdminAuthRequest hook is triggered on each successful API Admin
	// authentication request (sign-in, token refresh, etc.).
	//
	// Could be used to additionally validate or modify the
	// authenticated admin data and token.
	OnAdminAuthRequest() *hook.Hook[*AdminAuthEvent]

	// OnAdminBeforeAuthWithPasswordRequest hook is triggered before each Admin
	// auth with password API request (after request data load and before password validation).
	//
	// Could be used to implement for example a custom password validation
	// or to locate a different Admin identity (by assigning [AdminAuthWithPasswordEvent.Admin]).
	OnAdminBeforeAuthWithPasswordRequest() *hook.Hook[*AdminAuthWithPasswordEvent]

	// OnAdminAfterAuthWithPasswordRequest hook is triggered after each
	// successful Admin auth with password API request.
	OnAdminAfterAuthWithPasswordRequest() *hook.Hook[*AdminAuthWithPasswordEvent]

	// OnAdminBeforeAuthRefreshRequest hook is triggered before each Admin
	// auth refresh API request (right before generating a new auth token).
	//
	// Could be used to additionally validate the request data or implement
	// completely different auth refresh behavior.
	OnAdminBeforeAuthRefreshRequest() *hook.Hook[*AdminAuthRefreshEvent]

	// OnAdminAfterAuthRefreshRequest hook is triggered after each
	// successful auth refresh API request (right after generating a new auth token).
	OnAdminAfterAuthRefreshRequest() *hook.Hook[*AdminAuthRefreshEvent]

	// OnAdminBeforeRequestPasswordResetRequest hook is triggered before each Admin
	// request password reset API request (after request data load and before sending the reset email).
	//
	// Could be used to additionally validate the request data or implement
	// completely different password reset behavior.
	OnAdminBeforeRequestPasswordResetRequest() *hook.Hook[*AdminRequestPasswordResetEvent]

	// OnAdminAfterRequestPasswordResetRequest hook is triggered after each
	// successful request password reset API request.
	OnAdminAfterRequestPasswordResetRequest() *hook.Hook[*AdminRequestPasswordResetEvent]

	// OnAdminBeforeConfirmPasswordResetRequest hook is triggered before each Admin
	// confirm password reset API request (after request data load and before persistence).
	//
	// Could be used to additionally validate the request data or implement
	// completely different persistence behavior.
	OnAdminBeforeConfirmPasswordResetRequest() *hook.Hook[*AdminConfirmPasswordResetEvent]

	// OnAdminAfterConfirmPasswordResetRequest hook is triggered after each
	// successful confirm password reset API request.
	OnAdminAfterConfirmPasswordResetRequest() *hook.Hook[*AdminConfirmPasswordResetEvent]

	// ---------------------------------------------------------------
	// Record CRUD API event hooks
	// ---------------------------------------------------------------

	// OnRecordsListRequest hook is triggered on each API Records list request.
	//
	// Could be used to validate or modify the response before returning it to the client.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordsListRequest(tags ...string) *hook.TaggedHook[*RecordsListEvent]

	// OnRecordViewRequest hook is triggered on each API Record view request.
	//
	// Could be used to validate or modify the response before returning it to the client.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordViewRequest(tags ...string) *hook.TaggedHook[*RecordViewEvent]

	// OnRecordBeforeCreateRequest hook is triggered before each API Record
	// create request (after request data load and before model persistence).
	//
	// Could be used to additionally validate the request data or implement
	// completely different persistence behavior.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordBeforeCreateRequest(tags ...string) *hook.TaggedHook[*RecordCreateEvent]

	// OnRecordAfterCreateRequest hook is triggered after each
	// successful API Record create request.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordAfterCreateRequest(tags ...string) *hook.TaggedHook[*RecordCreateEvent]

	// OnRecordBeforeUpdateRequest hook is triggered before each API Record
	// update request (after request data load and before model persistence).
	//
	// Could be used to additionally validate the request data or implement
	// completely different persistence behavior.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordBeforeUpdateRequest(tags ...string) *hook.TaggedHook[*RecordUpdateEvent]

	// OnRecordAfterUpdateRequest hook is triggered after each
	// successful API Record update request.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordAfterUpdateRequest(tags ...string) *hook.TaggedHook[*RecordUpdateEvent]

	// OnRecordBeforeDeleteRequest hook is triggered before each API Record
	// delete request (after model load and before actual deletion).
	//
	// Could be used to additionally validate the request data or implement
	// completely different delete behavior.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordBeforeDeleteRequest(tags ...string) *hook.TaggedHook[*RecordDeleteEvent]

	// OnRecordAfterDeleteRequest hook is triggered after each
	// successful API Record delete request.
	//
	// If the optional "tags" list (Collection ids or names) is specified,
	// then all event handlers registered via the created hook will be
	// triggered and called only if their event data origin matches the tags.
	OnRecordAfterDeleteRequest(tags ...string) *hook.TaggedHook[*RecordDeleteEvent]

	// ---------------------------------------------------------------
	// Collection API event hooks
	// ---------------------------------------------------------------

	// OnCollectionsListRequest hook is triggered on each API Collections list request.
	//
	// Could be used to validate or modify the response before returning it to the client.
	OnCollectionsListRequest() *hook.Hook[*CollectionsListEvent]

	// OnCollectionViewRequest hook is triggered on each API Collection view request.
	//
	// Could be used to validate or modify the response before returning it to the client.
	OnCollectionViewRequest() *hook.Hook[*CollectionViewEvent]

	// OnCollectionBeforeCreateRequest hook is triggered before each API Collection
	// create request (after request data load and before model persistence).
	//
	// Could be used to additionally validate the request data or implement
	// completely different persistence behavior.
	OnCollectionBeforeCreateRequest() *hook.Hook[*CollectionCreateEvent]

	// OnCollectionAfterCreateRequest hook is triggered after each
	// successful API Collection create request.
	OnCollectionAfterCreateRequest() *hook.Hook[*CollectionCreateEvent]

	// OnCollectionBeforeUpdateRequest hook is triggered before each API Collection
	// update request (after request data load and before model persistence).
	//
	// Could be used to additionally validate the request data or implement
	// completely different persistence behavior.
	OnCollectionBeforeUpdateRequest() *hook.Hook[*CollectionUpdateEvent]

	// OnCollectionAfterUpdateRequest hook is triggered after each
	// successful API Collection update request.
	OnCollectionAfterUpdateRequest() *hook.Hook[*CollectionUpdateEvent]

	// OnCollectionBeforeDeleteRequest hook is triggered before each API
	// Collection delete request (after model load and before actual deletion).
	//
	// Could be used to additionally validate the request data or implement
	// completely different delete behavior.
	OnCollectionBeforeDeleteRequest() *hook.Hook[*CollectionDeleteEvent]

	// OnCollectionAfterDeleteRequest hook is triggered after each
	// successful API Collection delete request.
	OnCollectionAfterDeleteRequest() *hook.Hook[*CollectionDeleteEvent]

	// OnCollectionsBeforeImportRequest hook is triggered before each API
	// collections import request (after request data load and before the actual import).
	//
	// Could be used to additionally validate the imported collections or
	// to implement completely different import behavior.
	OnCollectionsBeforeImportRequest() *hook.Hook[*CollectionsImportEvent]

	// OnCollectionsAfterImportRequest hook is triggered after each
	// successful API collections import request.
	OnCollectionsAfterImportRequest() *hook.Hook[*CollectionsImportEvent]
}
