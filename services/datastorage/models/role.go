package models

// Role represents the database model of roles
type Role struct {
	BaseModel

	Name        string `gorm:"size:255;not null" json:"name"`
	GuardName   string `gorm:"size:255;not null;index" json:"guard_name"`
	Description string `gorm:"size:255;" json:"description"`

	// Many to Many
	Permissions []Permission `gorm:"many2many:role_permissions;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"permissions"`
}

// TableName sets the table name
func (Role) TableName() string {
	return "roles"
}
