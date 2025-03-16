package models

import (
	"time"

	"gorm.io/gorm"
)

// Gun represents a firearm in the system
type Gun struct {
	gorm.Model
	Name           string
	Description    string
	SerialNumber   string
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
	return db.Save(gun).Error
}

// DeleteGun deletes a gun from the database
func DeleteGun(db *gorm.DB, id uint, ownerID uint) error {
	return db.Where("id = ? AND owner_id = ?", id, ownerID).Delete(&Gun{}).Error
}
