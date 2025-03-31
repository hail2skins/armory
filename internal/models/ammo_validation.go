package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrAmmoNameRequired is returned when an ammo name is empty
	ErrAmmoNameRequired = errors.New("ammo name is required")

	// ErrAmmoNameTooLong is returned when an ammo name exceeds the maximum allowed length
	ErrAmmoNameTooLong = errors.New("ammo name exceeds maximum length of 100 characters")

	// ErrAmmoNegativePrice is returned when an ammo's paid value is negative
	ErrAmmoNegativePrice = errors.New("ammo price cannot be negative")

	// ErrAmmoNegativeCount is returned when an ammo's count is negative
	ErrAmmoNegativeCount = errors.New("ammo count cannot be negative")

	// ErrAmmoFutureDate is returned when the acquired date is in the future
	ErrAmmoFutureDate = errors.New("acquired date cannot be in the future")

	// ErrInvalidBrand is returned when the brand ID doesn't exist
	ErrInvalidBrand = errors.New("invalid brand ID")

	// ErrInvalidBulletStyle is returned when the bullet style ID doesn't exist
	ErrInvalidBulletStyle = errors.New("invalid bullet style ID")

	// ErrInvalidGrain is returned when the grain ID doesn't exist
	ErrInvalidGrain = errors.New("invalid grain ID")

	// ErrInvalidCasing is returned when the casing ID doesn't exist
	ErrInvalidCasing = errors.New("invalid casing ID")
)

// Note: ErrInvalidCaliber is imported from gun_validation.go

// Validate validates the Ammo model
func (a *Ammo) Validate(db *gorm.DB) error {
	// Validate name is not empty
	if a.Name == "" {
		return ErrAmmoNameRequired
	}

	// Validate name length (max 100 characters)
	if len(a.Name) > 100 {
		return ErrAmmoNameTooLong
	}

	// Validate paid (can't be negative)
	if a.Paid != nil && *a.Paid < 0 {
		return ErrAmmoNegativePrice
	}

	// Validate count (can't be negative)
	if a.Count < 0 {
		return ErrAmmoNegativeCount
	}

	// Validate acquired date (can't be in the future)
	if a.Acquired != nil && a.Acquired.After(time.Now()) {
		return ErrAmmoFutureDate
	}

	// Validate foreign keys if db is provided
	if db != nil {
		// Check BrandID
		var count int64
		if err := db.Model(&Brand{}).Where("id = ?", a.BrandID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return ErrInvalidBrand
		}

		// Check CaliberID
		if err := db.Model(&Caliber{}).Where("id = ?", a.CaliberID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return ErrInvalidCaliber
		}

		// Check BulletStyleID (optional)
		if a.BulletStyleID > 0 {
			if err := db.Model(&BulletStyle{}).Where("id = ?", a.BulletStyleID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalidBulletStyle
			}
		}

		// Check GrainID (optional)
		if a.GrainID > 0 {
			if err := db.Model(&Grain{}).Where("id = ?", a.GrainID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalidGrain
			}
		}

		// Check CasingID (optional)
		if a.CasingID > 0 {
			if err := db.Model(&Casing{}).Where("id = ?", a.CasingID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalidCasing
			}
		}
	}

	return nil
}

// CreateAmmoWithValidation creates a new ammo record in the database with validation
func CreateAmmoWithValidation(db *gorm.DB, ammo *Ammo) error {
	// Validate the ammo
	if err := ammo.Validate(db); err != nil {
		return err
	}

	// Create the ammo
	return db.Create(ammo).Error
}

// UpdateAmmoWithValidation updates an existing ammo record in the database with validation
func UpdateAmmoWithValidation(db *gorm.DB, ammo *Ammo) error {
	// Validate the ammo
	if err := ammo.Validate(db); err != nil {
		return err
	}

	// Set the updated_at time
	ammo.UpdatedAt = time.Now()

	// First, retrieve the existing ammo to ensure we're working with the latest data
	var existingAmmo Ammo
	if err := db.First(&existingAmmo, ammo.ID).Error; err != nil {
		return err
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
