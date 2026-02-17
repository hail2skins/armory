package models

import (
	"time"

	"gorm.io/gorm"
)

// Range represents a shooting range location
type Range struct {
	gorm.Model
	RangeName    string
	StreetNumber string
	StreetName   string
	AddressLine2 string
	City         string
	State        string `gorm:"size:2"`
	Zip          string `gorm:"size:5"`
}

// TableName specifies the table name for the Range model
func (Range) TableName() string {
	return "ranges"
}

// FindAllRanges retrieves all ranges from the database
func FindAllRanges(db *gorm.DB) ([]Range, error) {
	var ranges []Range
	if err := db.Find(&ranges).Error; err != nil {
		return nil, err
	}
	return ranges, nil
}

// FindRangeByID retrieves a range by its ID
func FindRangeByID(db *gorm.DB, id uint) (*Range, error) {
	var r Range
	if err := db.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRange creates a new range in the database
func CreateRange(db *gorm.DB, r *Range) error {
	return db.Create(r).Error
}

// UpdateRange updates an existing range in the database
func UpdateRange(db *gorm.DB, r *Range) error {
	r.UpdatedAt = time.Now()

	var existingRange Range
	if err := db.First(&existingRange, r.ID).Error; err != nil {
		return err
	}

	result := db.Model(&existingRange).Updates(map[string]interface{}{
		"range_name":    r.RangeName,
		"street_number": r.StreetNumber,
		"street_name":   r.StreetName,
		"address_line2": r.AddressLine2,
		"city":          r.City,
		"state":         r.State,
		"zip":           r.Zip,
		"updated_at":    r.UpdatedAt,
	})
	if result.Error != nil {
		return result.Error
	}

	return db.First(r, r.ID).Error
}

// DeleteRange deletes a range from the database
func DeleteRange(db *gorm.DB, id uint) error {
	var r Range
	if err := db.Where("id = ?", id).First(&r).Error; err != nil {
		return err
	}

	return db.Delete(&r).Error
}
