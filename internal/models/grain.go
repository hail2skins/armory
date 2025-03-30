package models

import "gorm.io/gorm"

// Grain represents the weight of a projectile in grains.
type Grain struct {
	gorm.Model
	Weight     int `gorm:"unique;not null"` // Grain weight (e.g., 115, 55)
	Popularity int `gorm:"not null;default:0"`
	// Add any other relevant fields if needed
}

// FindAllGrains retrieves all grain weights from the database, ordered by Popularity and Weight
func FindAllGrains(db *gorm.DB) ([]Grain, error) {
	var grains []Grain
	// Order by Popularity descending, then by Weight ascending for consistent ordering
	if err := db.Order("popularity DESC, weight ASC").Find(&grains).Error; err != nil {
		return nil, err
	}
	return grains, nil
}

// FindGrainByID retrieves a grain by its ID
func FindGrainByID(db *gorm.DB, id uint) (*Grain, error) {
	var grain Grain
	if err := db.First(&grain, id).Error; err != nil {
		return nil, err
	}
	return &grain, nil
}

// FindGrainByWeight retrieves a grain by its Weight
func FindGrainByWeight(db *gorm.DB, weight int) (*Grain, error) {
	var grain Grain
	if err := db.Where("weight = ?", weight).First(&grain).Error; err != nil {
		return nil, err
	}
	return &grain, nil
}

// CreateGrain creates a new grain in the database
func CreateGrain(db *gorm.DB, grain *Grain) error {
	return db.Create(grain).Error
}

// UpdateGrain updates an existing grain in the database
func UpdateGrain(db *gorm.DB, grain *Grain) error {
	return db.Save(grain).Error
}

// DeleteGrain deletes a grain from the database
func DeleteGrain(db *gorm.DB, id uint) error {
	return db.Delete(&Grain{}, id).Error
}
