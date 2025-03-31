package models

import "gorm.io/gorm"

// Brand represents an ammunition brand/manufacturer.
type Brand struct {
	gorm.Model
	Name       string `gorm:"unique;not null"` // Full name of the brand
	Nickname   string // Common abbreviation or nickname
	Popularity int    `gorm:"not null;default:0"`
	// Add other fields like Country, Website, etc. if desired later
}

// GetID returns the brand's ID
func (b *Brand) GetID() uint {
	return b.ID
}

// GetName returns the brand's name
func (b *Brand) GetName() string {
	return b.Name
}

// GetNickname returns the brand's nickname
func (b *Brand) GetNickname() string {
	return b.Nickname
}

// GetPopularity returns the brand's popularity
func (b *Brand) GetPopularity() int {
	return b.Popularity
}

// SetName sets the brand's name
func (b *Brand) SetName(name string) {
	b.Name = name
}

// SetNickname sets the brand's nickname
func (b *Brand) SetNickname(nickname string) {
	b.Nickname = nickname
}

// SetPopularity sets the brand's popularity
func (b *Brand) SetPopularity(popularity int) {
	b.Popularity = popularity
}

// FindAllBrands retrieves all brands from the database
func FindAllBrands(db *gorm.DB) ([]Brand, error) {
	var brands []Brand
	if err := db.Order("name").Find(&brands).Error; err != nil {
		return nil, err
	}
	return brands, nil
}

// FindBrandByID retrieves a brand by its ID
func FindBrandByID(db *gorm.DB, id uint) (*Brand, error) {
	var brand Brand
	if err := db.First(&brand, id).Error; err != nil {
		return nil, err
	}
	return &brand, nil
}

// FindBrandByName retrieves a brand by its Name
func FindBrandByName(db *gorm.DB, name string) (*Brand, error) {
	var brand Brand
	if err := db.Where("name = ?", name).First(&brand).Error; err != nil {
		return nil, err
	}
	return &brand, nil
}

// CreateBrand creates a new brand in the database
func CreateBrand(db *gorm.DB, brand *Brand) error {
	return db.Create(brand).Error
}

// UpdateBrand updates an existing brand in the database
func UpdateBrand(db *gorm.DB, brand *Brand) error {
	return db.Save(brand).Error
}

// DeleteBrand deletes a brand from the database
func DeleteBrand(db *gorm.DB, id uint) error {
	return db.Delete(&Brand{}, id).Error
}

// GetAllBrands returns all brands from the database
func GetAllBrands(db *gorm.DB) ([]Brand, error) {
	var brands []Brand
	result := db.Find(&brands)
	if result.Error != nil {
		return nil, result.Error
	}
	return brands, nil
}
