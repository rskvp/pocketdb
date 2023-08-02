package models

import (
	"encoding/json"

	"done/services/datastorage/models/schema"
	"done/tools/types"
)

var (
	_ Model        = (*Collection)(nil)
	_ FilesManager = (*Collection)(nil)
)

const (
	CollectionTypeBase = "base"
	CollectionTypeAuth = "auth"
	CollectionTypeView = "view"
)

type Collection struct {
	BaseModel

	Name    string                  `db:"name" json:"name"`
	Type    string                  `db:"type" json:"type"`
	System  bool                    `db:"system" json:"system"`
	Schema  schema.Schema           `db:"schema" json:"schema"`
	Indexes types.JsonArray[string] `db:"indexes" json:"indexes"`

	// rules
	ListRule   *string `db:"listRule" json:"listRule"`
	ViewRule   *string `db:"viewRule" json:"viewRule"`
	CreateRule *string `db:"createRule" json:"createRule"`
	UpdateRule *string `db:"updateRule" json:"updateRule"`
	DeleteRule *string `db:"deleteRule" json:"deleteRule"`

	Options types.JsonMap `db:"options" json:"options"`
}

// TableName returns the Collection model SQL table name.
func (m *Collection) TableName() string {
	return "_collections"
}

// BaseFilesPath returns the storage dir path used by the collection.
func (m *Collection) BaseFilesPath() string {
	return m.Id
}

// IsBase checks if the current collection has "base" type.
func (m *Collection) IsBase() bool {
	return m.Type == CollectionTypeBase
}

// IsAuth checks if the current collection has "auth" type.
func (m *Collection) IsAuth() bool {
	return m.Type == CollectionTypeAuth
}

// IsView checks if the current collection has "view" type.
func (m *Collection) IsView() bool {
	return m.Type == CollectionTypeView
}

// MarshalJSON implements the [json.Marshaler] interface.
func (m Collection) MarshalJSON() ([]byte, error) {
	type alias Collection // prevent recursion

	m.NormalizeOptions()

	return json.Marshal(alias(m))
}

// BaseOptions decodes the current collection options and returns them
// as new [CollectionBaseOptions] instance.
func (m *Collection) BaseOptions() CollectionBaseOptions {
	result := CollectionBaseOptions{}
	m.DecodeOptions(&result)
	return result
}

// NormalizeOptions updates the current collection options with a
// new normalized state based on the collection type.
func (m *Collection) NormalizeOptions() error {
	var typedOptions = m.BaseOptions()

	// serialize
	raw, err := json.Marshal(typedOptions)
	if err != nil {
		return err
	}

	// load into a new JsonMap
	m.Options = types.JsonMap{}
	if err := json.Unmarshal(raw, &m.Options); err != nil {
		return err
	}

	return nil
}

// DecodeOptions decodes the current collection options into the
// provided "result" (must be a pointer).
func (m *Collection) DecodeOptions(result any) error {
	// raw serialize
	raw, err := json.Marshal(m.Options)
	if err != nil {
		return err
	}

	// decode into the provided result
	if err := json.Unmarshal(raw, result); err != nil {
		return err
	}

	return nil
}

// SetOptions normalizes and unmarshals the specified options into m.Options.
func (m *Collection) SetOptions(typedOptions any) error {
	// serialize
	raw, err := json.Marshal(typedOptions)
	if err != nil {
		return err
	}

	m.Options = types.JsonMap{}
	if err := json.Unmarshal(raw, &m.Options); err != nil {
		return err
	}

	return m.NormalizeOptions()
}

// -------------------------------------------------------------------

// CollectionBaseOptions defines the "base" Collection.Options fields.
type CollectionBaseOptions struct {
}

// Validate implements [validation.Validatable] interface.
func (o CollectionBaseOptions) Validate() error {
	return nil
}
