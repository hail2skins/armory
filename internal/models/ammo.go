package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Ammo represents ammunition in the system
type Ammo struct {
	gorm.Model
	Name          string     `gorm:"not null"`
	Acquired      *time.Time // When the ammo was acquired
	BrandID       uint       `gorm:"not null"`
	Brand         Brand      `gorm:"foreignKey:BrandID"`
	BulletStyleID uint
	BulletStyle   BulletStyle `gorm:"foreignKey:BulletStyleID"`
	GrainID       uint
	Grain         Grain   `gorm:"foreignKey:GrainID"`
	CaliberID     uint    `gorm:"not null"`
	Caliber       Caliber `gorm:"foreignKey:CaliberID"`
	CasingID      uint
	Casing        Casing      `gorm:"foreignKey:CasingID"`
	OwnerID       uint        `gorm:"not null"`
	Owner         interface{} `gorm:"-"` // This will be populated by the application, not stored in DB
	Paid          *float64    // Optional field for the price paid (in USD)
	Count         int         // Number of ammunition rounds
	Expended      int         // Number of rounds that have been used
	HasMoreAmmo   bool        `gorm:"-"` // Indicates if there are more ammo not being shown (not stored in DB)
	TotalAmmo     int         `gorm:"-"` // Total number of ammo the user has (not stored in DB)
}

// TableName specifies the table name for the Ammo model
func (Ammo) TableName() string {
	return "ammo"
}

// FindAmmoByOwner retrieves all ammo belonging to a specific owner
func FindAmmoByOwner(db *gorm.DB, ownerID uint) ([]Ammo, error) {
	// Get all ammo for this owner
	var allAmmo []Ammo
	if err := db.Preload("Brand").Preload("BulletStyle").Preload("Grain").
		Preload("Caliber").Preload("Casing").
		Where("owner_id = ?", ownerID).Find(&allAmmo).Error; err != nil {
		return nil, err
	}

	// Return all ammo
	return allAmmo, nil
}

// FindAmmoByID retrieves ammo by its ID, ensuring it belongs to the specified owner
func FindAmmoByID(db *gorm.DB, id uint, ownerID uint) (*Ammo, error) {
	var ammo Ammo
	if err := db.Preload("Brand").Preload("BulletStyle").Preload("Grain").
		Preload("Caliber").Preload("Casing").
		Where("id = ? AND owner_id = ?", id, ownerID).First(&ammo).Error; err != nil {
		return nil, err
	}
	return &ammo, nil
}

// CreateAmmo creates a new ammo record in the database
func CreateAmmo(db *gorm.DB, ammo *Ammo) error {
	// Validate expended count
	if ammo.Expended < 0 {
		return errors.New("expended count cannot be negative")
	}
	if ammo.Expended > ammo.Count {
		return errors.New("expended count cannot be greater than total count")
	}

	return db.Create(ammo).Error
}

// UpdateAmmo updates an existing ammo record in the database
func UpdateAmmo(db *gorm.DB, ammo *Ammo) error {
	// Set the updated_at time
	ammo.UpdatedAt = time.Now()

	// First, retrieve the existing ammo to ensure we're working with the latest data
	var existingAmmo Ammo
	if err := db.First(&existingAmmo, ammo.ID).Error; err != nil {
		return err
	}

	// Validate expended count
	if ammo.Expended < 0 {
		return errors.New("expended count cannot be negative")
	}
	if ammo.Expended > ammo.Count {
		return errors.New("expended count cannot be greater than total count")
	}

	// Update the ammo with all fields
	result := db.Model(&existingAmmo).Updates(map[string]interface{}{
		"name":            ammo.Name,
		"acquired":        ammo.Acquired,
		"brand_id":        ammo.BrandID,
		"bullet_style_id": ammo.BulletStyleID,
		"grain_id":        ammo.GrainID,
		"caliber_id":      ammo.CaliberID,
		"casing_id":       ammo.CasingID,
		"updated_at":      ammo.UpdatedAt,
		"paid":            ammo.Paid,
		"count":           ammo.Count,
		"expended":        ammo.Expended,
	})

	if result.Error != nil {
		return result.Error
	}

	// Reload the ammo with its relationships
	if err := db.Preload("Brand").Preload("BulletStyle").Preload("Grain").
		Preload("Caliber").Preload("Casing").
		First(ammo, ammo.ID).Error; err != nil {
		return err
	}

	return nil
}

// DeleteAmmo deletes ammo from the database
func DeleteAmmo(db *gorm.DB, id uint, ownerID uint) error {
	// First check if the ammo exists and belongs to the specified owner
	var ammo Ammo
	if err := db.Where("id = ?", id).First(&ammo).Error; err != nil {
		return err
	}

	// Verify ownership
	if ammo.OwnerID != ownerID {
		return errors.New("not authorized: ammo does not belong to this owner")
	}

	// If all checks pass, delete the ammo
	return db.Delete(&ammo).Error
}
