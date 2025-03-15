package database

import (
	"github.com/hail2skins/armory/internal/models"
)

// AdminService defines the interface for admin-related operations
type AdminService interface {
	// Manufacturer operations
	FindAllManufacturers() ([]models.Manufacturer, error)
	FindManufacturerByID(id uint) (*models.Manufacturer, error)
	CreateManufacturer(manufacturer *models.Manufacturer) error
	UpdateManufacturer(manufacturer *models.Manufacturer) error
	DeleteManufacturer(id uint) error

	// Caliber operations
	FindAllCalibers() ([]models.Caliber, error)
	FindCaliberByID(id uint) (*models.Caliber, error)
	CreateCaliber(caliber *models.Caliber) error
	UpdateCaliber(caliber *models.Caliber) error
	DeleteCaliber(id uint) error

	// Weapon type operations
	FindAllWeaponTypes() ([]models.WeaponType, error)
	FindWeaponTypeByID(id uint) (*models.WeaponType, error)
	CreateWeaponType(weaponType *models.WeaponType) error
	UpdateWeaponType(weaponType *models.WeaponType) error
	DeleteWeaponType(id uint) error
}

// Ensure service implements AdminService
var _ AdminService = (*service)(nil)

// FindAllManufacturers retrieves all manufacturers
func (s *service) FindAllManufacturers() ([]models.Manufacturer, error) {
	var manufacturers []models.Manufacturer
	if err := s.db.Find(&manufacturers).Error; err != nil {
		return nil, err
	}
	return manufacturers, nil
}

// FindManufacturerByID retrieves a manufacturer by ID
func (s *service) FindManufacturerByID(id uint) (*models.Manufacturer, error) {
	var manufacturer models.Manufacturer
	if err := s.db.First(&manufacturer, id).Error; err != nil {
		return nil, err
	}
	return &manufacturer, nil
}

// CreateManufacturer creates a new manufacturer
func (s *service) CreateManufacturer(manufacturer *models.Manufacturer) error {
	return s.db.Create(manufacturer).Error
}

// UpdateManufacturer updates a manufacturer
func (s *service) UpdateManufacturer(manufacturer *models.Manufacturer) error {
	return s.db.Save(manufacturer).Error
}

// DeleteManufacturer deletes a manufacturer
func (s *service) DeleteManufacturer(id uint) error {
	return s.db.Delete(&models.Manufacturer{}, id).Error
}

// FindAllCalibers retrieves all calibers
func (s *service) FindAllCalibers() ([]models.Caliber, error) {
	var calibers []models.Caliber
	if err := s.db.Find(&calibers).Error; err != nil {
		return nil, err
	}
	return calibers, nil
}

// FindCaliberByID retrieves a caliber by ID
func (s *service) FindCaliberByID(id uint) (*models.Caliber, error) {
	var caliber models.Caliber
	if err := s.db.First(&caliber, id).Error; err != nil {
		return nil, err
	}
	return &caliber, nil
}

// CreateCaliber creates a new caliber
func (s *service) CreateCaliber(caliber *models.Caliber) error {
	return s.db.Create(caliber).Error
}

// UpdateCaliber updates a caliber
func (s *service) UpdateCaliber(caliber *models.Caliber) error {
	return s.db.Save(caliber).Error
}

// DeleteCaliber deletes a caliber
func (s *service) DeleteCaliber(id uint) error {
	return s.db.Delete(&models.Caliber{}, id).Error
}

// FindAllWeaponTypes retrieves all weapon types
func (s *service) FindAllWeaponTypes() ([]models.WeaponType, error) {
	var weaponTypes []models.WeaponType
	if err := s.db.Find(&weaponTypes).Error; err != nil {
		return nil, err
	}
	return weaponTypes, nil
}

// FindWeaponTypeByID retrieves a weapon type by ID
func (s *service) FindWeaponTypeByID(id uint) (*models.WeaponType, error) {
	var weaponType models.WeaponType
	if err := s.db.First(&weaponType, id).Error; err != nil {
		return nil, err
	}
	return &weaponType, nil
}

// CreateWeaponType creates a new weapon type
func (s *service) CreateWeaponType(weaponType *models.WeaponType) error {
	return s.db.Create(weaponType).Error
}

// UpdateWeaponType updates a weapon type
func (s *service) UpdateWeaponType(weaponType *models.WeaponType) error {
	return s.db.Save(weaponType).Error
}

// DeleteWeaponType deletes a weapon type
func (s *service) DeleteWeaponType(id uint) error {
	return s.db.Delete(&models.WeaponType{}, id).Error
}
