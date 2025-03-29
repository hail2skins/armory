package models

import "gorm.io/gorm"

// BulletStyle represents a type of bullet style in the system
type BulletStyle struct {
	gorm.Model
	Type       string `gorm:"unique;not null"`
	Nickname   string // Nickname or abbreviation
	Popularity int    `gorm:"not null;default:0"`
}

// FindAllBulletStyles retrieves all bullet styles from the database, ordered by Popularity and Type
func FindAllBulletStyles(db *gorm.DB) ([]BulletStyle, error) {
	var bulletStyles []BulletStyle
	// Order by Popularity descending, then by Type ascending for consistent ordering
	if err := db.Order("popularity DESC, type ASC").Find(&bulletStyles).Error; err != nil {
		return nil, err
	}
	return bulletStyles, nil
}

// FindBulletStyleByID retrieves a bullet style by its ID
func FindBulletStyleByID(db *gorm.DB, id uint) (*BulletStyle, error) {
	var bulletStyle BulletStyle
	if err := db.First(&bulletStyle, id).Error; err != nil {
		return nil, err
	}
	return &bulletStyle, nil
}

// FindBulletStyleByType retrieves a bullet style by its Type
func FindBulletStyleByType(db *gorm.DB, bulletStyleType string) (*BulletStyle, error) {
	var bulletStyle BulletStyle
	if err := db.Where("type = ?", bulletStyleType).First(&bulletStyle).Error; err != nil {
		return nil, err
	}
	return &bulletStyle, nil
}

// CreateBulletStyle creates a new bullet style in the database
func CreateBulletStyle(db *gorm.DB, bulletStyle *BulletStyle) error {
	return db.Create(bulletStyle).Error
}

// UpdateBulletStyle updates an existing bullet style in the database
func UpdateBulletStyle(db *gorm.DB, bulletStyle *BulletStyle) error {
	return db.Save(bulletStyle).Error
}

// DeleteBulletStyle deletes a bullet style from the database
func DeleteBulletStyle(db *gorm.DB, id uint) error {
	return db.Delete(&BulletStyle{}, id).Error
}
