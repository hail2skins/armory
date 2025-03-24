package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrGunNameTooLong is returned when a gun name exceeds the maximum allowed length
	ErrGunNameTooLong = errors.New("gun name exceeds maximum length of 100 characters")

	// ErrNegativePrice is returned when a gun's paid value is negative
	ErrNegativePrice = errors.New("gun price cannot be negative")

	// ErrFutureDate is returned when the acquired date is in the future
	ErrFutureDate = errors.New("acquired date cannot be in the future")

	// ErrInvalidWeaponType is returned when the weapon type ID doesn't exist
	ErrInvalidWeaponType = errors.New("invalid weapon type ID")

	// ErrInvalidCaliber is returned when the caliber ID doesn't exist
	ErrInvalidCaliber = errors.New("invalid caliber ID")

	// ErrInvalidManufacturer is returned when the manufacturer ID doesn't exist
	ErrInvalidManufacturer = errors.New("invalid manufacturer ID")
)

// Validate validates the Gun model
func (g *Gun) Validate(db *gorm.DB) error {
	// Validate name length (max 100 characters)
	if len(g.Name) > 100 {
		return ErrGunNameTooLong
	}

	// Validate paid (can't be negative)
	if g.Paid != nil && *g.Paid < 0 {
		return ErrNegativePrice
	}

	// Validate acquired date (can't be in the future)
	if g.Acquired != nil && g.Acquired.After(time.Now()) {
		return ErrFutureDate
	}

	// Validate foreign keys if db is provided
	if db != nil {
		// Check WeaponTypeID
		if g.WeaponTypeID > 0 {
			var count int64
			if err := db.Model(&WeaponType{}).Where("id = ?", g.WeaponTypeID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalidWeaponType
			}
		}

		// Check CaliberID
		if g.CaliberID > 0 {
			var count int64
			if err := db.Model(&Caliber{}).Where("id = ?", g.CaliberID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalidCaliber
			}
		}

		// Check ManufacturerID
		if g.ManufacturerID > 0 {
			var count int64
			if err := db.Model(&Manufacturer{}).Where("id = ?", g.ManufacturerID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrInvalidManufacturer
			}
		}
	}

	return nil
}

// Now add validation to the create and update functions

// CreateGunWithValidation creates a new gun in the database with validation
func CreateGunWithValidation(db *gorm.DB, gun *Gun) error {
	// Validate the gun
	if err := gun.Validate(db); err != nil {
		return err
	}

	// Create the gun
	return db.Create(gun).Error
}

// UpdateGunWithValidation updates an existing gun in the database with validation
func UpdateGunWithValidation(db *gorm.DB, gun *Gun) error {
	// Validate the gun
	if err := gun.Validate(db); err != nil {
		return err
	}

	// Set the updated_at time
	gun.UpdatedAt = time.Now()

	// First, retrieve the existing gun to ensure we're working with the latest data
	var existingGun Gun
	if err := db.First(&existingGun, gun.ID).Error; err != nil {
		return err
	}

	// Update the gun with all fields
	result := db.Model(&existingGun).Updates(map[string]interface{}{
		"name":            gun.Name,
		"serial_number":   gun.SerialNumber,
		"acquired":        gun.Acquired,
		"weapon_type_id":  gun.WeaponTypeID,
		"caliber_id":      gun.CaliberID,
		"manufacturer_id": gun.ManufacturerID,
		"updated_at":      gun.UpdatedAt,
		"paid":            gun.Paid,
	})

	if result.Error != nil {
		return result.Error
	}

	// Reload the gun with its relationships
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").First(gun, gun.ID).Error; err != nil {
		return err
	}

	return nil
}
