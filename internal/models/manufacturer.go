package models

import (
	"gorm.io/gorm"
)

// Manufacturer represents a firearm manufacturer in the system
type Manufacturer struct {
	gorm.Model
	Name       string `gorm:"not null"`
	Nickname   string
	Country    string `gorm:"not null"`
	Popularity int    `gorm:"default:0"` // Higher values appear first in dropdowns
}

// GetID returns the manufacturer's ID
func (m *Manufacturer) GetID() uint {
	return m.ID
}

// GetName returns the manufacturer's name
func (m *Manufacturer) GetName() string {
	return m.Name
}

// GetNickname returns the manufacturer's nickname
func (m *Manufacturer) GetNickname() string {
	return m.Nickname
}

// GetCountry returns the manufacturer's country
func (m *Manufacturer) GetCountry() string {
	return m.Country
}

// SetName sets the manufacturer's name
func (m *Manufacturer) SetName(name string) {
	m.Name = name
}

// SetNickname sets the manufacturer's nickname
func (m *Manufacturer) SetNickname(nickname string) {
	m.Nickname = nickname
}

// SetCountry sets the manufacturer's country
func (m *Manufacturer) SetCountry(country string) {
	m.Country = country
}

// FindAllManufacturers retrieves all manufacturers from the database
func FindAllManufacturers(db *gorm.DB) ([]Manufacturer, error) {
	var manufacturers []Manufacturer
	if err := db.Order("name").Find(&manufacturers).Error; err != nil {
		return nil, err
	}
	return manufacturers, nil
}

// FindManufacturerByID retrieves a manufacturer by its ID
func FindManufacturerByID(db *gorm.DB, id uint) (*Manufacturer, error) {
	var manufacturer Manufacturer
	if err := db.First(&manufacturer, id).Error; err != nil {
		return nil, err
	}
	return &manufacturer, nil
}

// CreateManufacturer creates a new manufacturer in the database
func CreateManufacturer(db *gorm.DB, manufacturer *Manufacturer) error {
	return db.Create(manufacturer).Error
}

// UpdateManufacturer updates an existing manufacturer in the database
func UpdateManufacturer(db *gorm.DB, manufacturer *Manufacturer) error {
	return db.Save(manufacturer).Error
}

// DeleteManufacturer deletes a manufacturer from the database
func DeleteManufacturer(db *gorm.DB, id uint) error {
	return db.Delete(&Manufacturer{}, id).Error
}
