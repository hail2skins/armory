package models

import (
	"gorm.io/gorm"
)

// Caliber represents an ammunition caliber in the system
type Caliber struct {
	gorm.Model
	Caliber    string `gorm:"size:100;not null;unique" json:"caliber"`
	Nickname   string `gorm:"size:50" json:"nickname"`
	Popularity int    `gorm:"default:0" json:"popularity"` // Higher values appear first in dropdowns
}

// FindAllCalibers retrieves all calibers from the database
func FindAllCalibers(db *gorm.DB) ([]Caliber, error) {
	var calibers []Caliber
	if err := db.Order("caliber").Find(&calibers).Error; err != nil {
		return nil, err
	}
	return calibers, nil
}

// FindCaliberByID retrieves a caliber by its ID
func FindCaliberByID(db *gorm.DB, id uint) (*Caliber, error) {
	var caliber Caliber
	if err := db.First(&caliber, id).Error; err != nil {
		return nil, err
	}
	return &caliber, nil
}

// CreateCaliber creates a new caliber in the database
func CreateCaliber(db *gorm.DB, caliber *Caliber) error {
	return db.Create(caliber).Error
}

// UpdateCaliber updates an existing caliber in the database
func UpdateCaliber(db *gorm.DB, caliber *Caliber) error {
	return db.Save(caliber).Error
}

// DeleteCaliber deletes a caliber from the database
func DeleteCaliber(db *gorm.DB, id uint) error {
	return db.Delete(&Caliber{}, id).Error
}
