package models

import (
	"gorm.io/gorm"
)

// Casing represents an ammunition casing type in the system
type Casing struct {
	gorm.Model
	Type       string `gorm:"size:50;not null;unique"`
	Popularity int    `gorm:"default:0"` // Higher values appear first in dropdowns
}

// FindAllCasings retrieves all casings from the database, ordered by Type
func FindAllCasings(db *gorm.DB) ([]Casing, error) {
	var casings []Casing
	// Order by Popularity descending, then by Type ascending for consistent ordering
	if err := db.Order("popularity DESC, type ASC").Find(&casings).Error; err != nil {
		return nil, err
	}
	return casings, nil
}

// FindCasingByID retrieves a casing by its ID
func FindCasingByID(db *gorm.DB, id uint) (*Casing, error) {
	var casing Casing
	if err := db.First(&casing, id).Error; err != nil {
		return nil, err
	}
	return &casing, nil
}

// FindCasingByType retrieves a casing by its Type
func FindCasingByType(db *gorm.DB, casingType string) (*Casing, error) {
	var casing Casing
	if err := db.Where("type = ?", casingType).First(&casing).Error; err != nil {
		return nil, err
	}
	return &casing, nil
}

// CreateCasing creates a new casing in the database
func CreateCasing(db *gorm.DB, casing *Casing) error {
	return db.Create(casing).Error
}

// UpdateCasing updates an existing casing in the database
func UpdateCasing(db *gorm.DB, casing *Casing) error {
	return db.Save(casing).Error
}

// DeleteCasing deletes a casing from the database
func DeleteCasing(db *gorm.DB, id uint) error {
	return db.Delete(&Casing{}, id).Error
}
