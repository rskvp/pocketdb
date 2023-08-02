package daos

import (
	"fmt"
	"strings"

	"done/tools/list"

	"done/services/datastorage/models"
	"done/services/datastorage/models/schema"

	"github.com/pocketbase/dbx"
)

// CollectionQuery returns a new Collection select query.
func (dao *Dao) CollectionQuery() *dbx.SelectQuery {
	return dao.ModelQuery(&models.Collection{})
}

// FindCollectionsByType finds all collections by the given type.
func (dao *Dao) FindCollectionsByType(collectionType string) ([]*models.Collection, error) {
	collections := []*models.Collection{}

	err := dao.CollectionQuery().
		AndWhere(dbx.HashExp{"type": collectionType}).
		OrderBy("created ASC").
		All(&collections)

	if err != nil {
		return nil, err
	}

	return collections, nil
}

// FindCollectionByNameOrId finds a single collection by its name (case insensitive) or id.
func (dao *Dao) FindCollectionByNameOrId(nameOrId string) (*models.Collection, error) {
	model := &models.Collection{}

	err := dao.CollectionQuery().
		AndWhere(dbx.NewExp("[[id]] = {:id} OR LOWER([[name]])={:name}", dbx.Params{
			"id":   nameOrId,
			"name": strings.ToLower(nameOrId),
		})).
		Limit(1).
		One(model)

	if err != nil {
		return nil, err
	}

	return model, nil
}

// IsCollectionNameUnique checks that there is no existing collection
// with the provided name (case insensitive!).
//
// Note: case insensitive check because the name is used also as a table name for the records.
func (dao *Dao) IsCollectionNameUnique(name string, excludeIds ...string) bool {
	if name == "" {
		return false
	}

	query := dao.CollectionQuery().
		Select("count(*)").
		AndWhere(dbx.NewExp("LOWER([[name]])={:name}", dbx.Params{"name": strings.ToLower(name)})).
		Limit(1)

	if uniqueExcludeIds := list.NonzeroUniques(excludeIds); len(uniqueExcludeIds) > 0 {
		query.AndWhere(dbx.NotIn("id", list.ToInterfaceSlice(uniqueExcludeIds)...))
	}

	var exists bool

	return query.Row(&exists) == nil && !exists
}

// FindCollectionReferences returns information for all
// relation schema fields referencing the provided collection.
//
// If the provided collection has reference to itself then it will be
// also included in the result. To exclude it, pass the collection id
// as the excludeId argument.
func (dao *Dao) FindCollectionReferences(collection *models.Collection, excludeIds ...string) (map[*models.Collection][]*schema.SchemaField, error) {
	collections := []*models.Collection{}

	query := dao.CollectionQuery()

	if uniqueExcludeIds := list.NonzeroUniques(excludeIds); len(uniqueExcludeIds) > 0 {
		query.AndWhere(dbx.NotIn("id", list.ToInterfaceSlice(uniqueExcludeIds)...))
	}

	if err := query.All(&collections); err != nil {
		return nil, err
	}

	result := map[*models.Collection][]*schema.SchemaField{}

	for _, c := range collections {
		for _, f := range c.Schema.Fields() {
			if f.Type != schema.FieldTypeRelation {
				continue
			}
			f.InitOptions()
			options, _ := f.Options.(*schema.RelationOptions)
			if options != nil && options.CollectionId == collection.Id {
				result[c] = append(result[c], f)
			}
		}
	}

	return result, nil
}

// DeleteCollection deletes the provided Collection model.
// This method automatically deletes the related collection records table.
//
// NB! The collection cannot be deleted, if:
// - is system collection (aka. collection.System is true)
// - is referenced as part of a relation field in another collection
func (dao *Dao) DeleteCollection(collection *models.Collection) error {
	if collection.System {
		return fmt.Errorf("System collection %q cannot be deleted.", collection.Name)
	}

	// ensure that there aren't any existing references.
	// note: the select is outside of the transaction to prevent SQLITE_LOCKED error when mixing read&write in a single transaction
	result, err := dao.FindCollectionReferences(collection, collection.Id)
	if err != nil {
		return err
	}
	if total := len(result); total > 0 {
		names := make([]string, 0, len(result))
		for ref := range result {
			names = append(names, ref.Name)
		}
		return fmt.Errorf("The collection %q has external relation field references (%s).", collection.Name, strings.Join(names, ", "))
	}

	return dao.RunInTransaction(func(txDao *Dao) error {
		// delete the related view or records table

		if err := txDao.DeleteTable(collection.Name); err != nil {
			return err
		}

		return txDao.Delete(collection)
	})
}

// SaveCollection persists the provided Collection model and updates
// its related records table schema.
//
// If collecction.IsNew() is true, the method will perform a create, otherwise an update.
// To explicitly mark a collection for update you can use collecction.MarkAsNotNew().
func (dao *Dao) SaveCollection(collection *models.Collection) error {
	var oldCollection *models.Collection

	if !collection.IsNew() {
		// get the existing collection state to compare with the new one
		// note: the select is outside of the transaction to prevent SQLITE_LOCKED error when mixing read&write in a single transaction
		var findErr error
		oldCollection, findErr = dao.FindCollectionByNameOrId(collection.Id)
		if findErr != nil {
			return findErr
		}
	}

	txErr := dao.RunInTransaction(func(txDao *Dao) error {
		// set default collection type
		if collection.Type == "" {
			collection.Type = models.CollectionTypeBase
		}

		// persist the collection model
		if err := txDao.Save(collection); err != nil {
			return err
		}

		// sync the changes with the related records table
		if err := txDao.SyncRecordTableSchema(collection, oldCollection); err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return txErr
	}

	return nil
}
