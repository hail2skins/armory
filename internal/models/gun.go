package models

import (
	"time"

	"errors"

	"gorm.io/gorm"
)

// Gun represents a firearm in the system
type Gun struct {
	gorm.Model
	Name           string
	SerialNumber   string
	Purpose        string // Purpose of the gun (e.g., "Carry", "Plinking", "Home Defense")
	Finish         string // Finish of the gun (e.g., "Bluing", "Cerakote", "Stainless", "Nickel Plating")
	Acquired       *time.Time
	WeaponTypeID   uint
	WeaponType     WeaponType `gorm:"foreignKey:WeaponTypeID"`
	CaliberID      uint
	Caliber        Caliber `gorm:"foreignKey:CaliberID"`
	ManufacturerID uint
	Manufacturer   Manufacturer `gorm:"foreignKey:ManufacturerID"`
	OwnerID        uint
	Owner          interface{} `gorm:"-"` // This will be populated by the application, not stored in DB
	HasMoreGuns    bool        `gorm:"-"` // Indicates if there are more guns not being shown (not stored in DB)
	TotalGuns      int         `gorm:"-"` // Total number of guns the user has (not stored in DB)
	Paid           *float64    // Optional field for the price paid (in USD)
}

// TableName specifies the table name for the Gun model
func (Gun) TableName() string {
	return "guns"
}

// FindGunsByOwner retrieves all guns belonging to a specific owner
// For free tier users, only returns the first 2 guns
func FindGunsByOwner(db *gorm.DB, ownerID uint) ([]Gun, error) {
	// Get all guns for this owner
	var allGuns []Gun
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("owner_id = ?", ownerID).Find(&allGuns).Error; err != nil {
		return nil, err
	}

	// Check if we need to apply free tier limits
	// This will be handled by the controller based on the user's subscription status
	return allGuns, nil
}

// FindGunByID retrieves a gun by its ID, ensuring it belongs to the specified owner
func FindGunByID(db *gorm.DB, id uint, ownerID uint) (*Gun, error) {
	var gun Gun
	if err := db.Preload("WeaponType").Preload("Caliber").Preload("Manufacturer").Where("id = ? AND owner_id = ?", id, ownerID).First(&gun).Error; err != nil {
		return nil, err
	}
	return &gun, nil
}

// CreateGun creates a new gun in the database
func CreateGun(db *gorm.DB, gun *Gun) error {
	return db.Create(gun).Error
}

// UpdateGun updates an existing gun in the database
func UpdateGun(db *gorm.DB, gun *Gun) error {
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
		"purpose":         gun.Purpose,
		"finish":          gun.Finish,
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

// DeleteGun deletes a gun from the database
func DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	// First check if the gun exists and belongs to the specified owner
	var gun Gun
	if err := db.Where("id = ?", id).First(&gun).Error; err != nil {
		return err
	}

	// Verify ownership
	if gun.OwnerID != ownerID {
		return errors.New("not authorized: gun does not belong to this owner")
	}

	// If all checks pass, delete the gun
	return db.Delete(&gun).Error
}
