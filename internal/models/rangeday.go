package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// RangeDay represents a user's visit to a range
type RangeDay struct {
	gorm.Model
	UserID     uint
	RangeID    uint
	Range      Range `gorm:"foreignKey:RangeID"`
	Date       time.Time
	Comments   string
	ShotsFired int
	GunID      uint
	Gun        Gun `gorm:"foreignKey:GunID"`
	AmmoID     uint
	Ammo       Ammo `gorm:"foreignKey:AmmoID"`
}

// TableName specifies the table name for the RangeDay model
func (RangeDay) TableName() string {
	return "range_days"
}

// FindRangeDaysByUser retrieves all range day records for a user
func FindRangeDaysByUser(db *gorm.DB, userID uint) ([]RangeDay, error) {
	var rangeDays []RangeDay
	if err := db.Preload("Range").Preload("Gun").Preload("Ammo").Where("user_id = ?", userID).Find(&rangeDays).Error; err != nil {
		return nil, err
	}
	return rangeDays, nil
}

// FindRangeDayByID retrieves a range day record by ID for a user
func FindRangeDayByID(db *gorm.DB, id uint, userID uint) (*RangeDay, error) {
	var rangeDay RangeDay
	if err := db.Preload("Range").Preload("Gun").Preload("Ammo").Where("id = ? AND user_id = ?", id, userID).First(&rangeDay).Error; err != nil {
		return nil, err
	}
	return &rangeDay, nil
}

// CreateRangeDay creates a new range day record
func CreateRangeDay(db *gorm.DB, rangeDay *RangeDay) error {
	return db.Create(rangeDay).Error
}

// UpdateRangeDay updates an existing range day record
func UpdateRangeDay(db *gorm.DB, rangeDay *RangeDay) error {
	rangeDay.UpdatedAt = time.Now()

	var existingRangeDay RangeDay
	if err := db.First(&existingRangeDay, rangeDay.ID).Error; err != nil {
		return err
	}

	result := db.Model(&existingRangeDay).Updates(map[string]interface{}{
		"user_id":     rangeDay.UserID,
		"range_id":    rangeDay.RangeID,
		"date":        rangeDay.Date,
		"comments":    rangeDay.Comments,
		"shots_fired": rangeDay.ShotsFired,
		"gun_id":      rangeDay.GunID,
		"ammo_id":     rangeDay.AmmoID,
		"updated_at":  rangeDay.UpdatedAt,
	})

	if result.Error != nil {
		return result.Error
	}

	return db.Preload("Range").Preload("Gun").Preload("Ammo").First(rangeDay, rangeDay.ID).Error
}

// DeleteRangeDay deletes a range day record
func DeleteRangeDay(db *gorm.DB, id uint, userID uint) error {
	var rangeDay RangeDay
	if err := db.Where("id = ?", id).First(&rangeDay).Error; err != nil {
		return err
	}

	if rangeDay.UserID != userID {
		return errors.New("not authorized: range day does not belong to this user")
	}

	return db.Delete(&rangeDay).Error
}
