package models

import (
	"time"

	"gorm.io/gorm"
)

// WeaponType represents a type of weapon (e.g., handgun, rifle, shotgun)
type WeaponType struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Type       string         `gorm:"not null;unique" json:"type"`
	Nickname   string         `json:"nickname"`
	Popularity int            `gorm:"default:0" json:"popularity"` // Higher values appear first in dropdowns
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the WeaponType model
func (WeaponType) TableName() string {
	return "weapon_types"
}

// FindAllWeaponTypes retrieves all weapon types from the database
func FindAllWeaponTypes(db *gorm.DB) ([]WeaponType, error) {
	var weaponTypes []WeaponType
	if err := db.Find(&weaponTypes).Error; err != nil {
		return nil, err
	}
	return weaponTypes, nil
}

// FindWeaponTypeByID retrieves a weapon type by its ID
func FindWeaponTypeByID(db *gorm.DB, id uint) (*WeaponType, error) {
	var weaponType WeaponType
	if err := db.First(&weaponType, id).Error; err != nil {
		return nil, err
	}
	return &weaponType, nil
}

// CreateWeaponType creates a new weapon type in the database
func CreateWeaponType(db *gorm.DB, weaponType *WeaponType) error {
	return db.Create(weaponType).Error
}

// UpdateWeaponType updates an existing weapon type in the database
func UpdateWeaponType(db *gorm.DB, weaponType *WeaponType) error {
	return db.Save(weaponType).Error
}

// DeleteWeaponType deletes a weapon type from the database
func DeleteWeaponType(db *gorm.DB, id uint) error {
	return db.Delete(&WeaponType{}, id).Error
}
