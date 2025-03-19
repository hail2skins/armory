package data

import (
	"github.com/hail2skins/armory/internal/models"
)

// AdminData contains data for admin views
type AdminData struct {
	AuthData

	// For manufacturers
	Manufacturers []models.Manufacturer
	Manufacturer  *models.Manufacturer

	// For calibers
	Calibers []models.Caliber
	Caliber  *models.Caliber

	// For weapon types
	WeaponTypes []models.WeaponType
	WeaponType  *models.WeaponType
}

// NewAdminData creates a new AdminData with default values
func NewAdminData() *AdminData {
	return &AdminData{
		AuthData: NewAuthData(),
	}
}

// WithTitle returns a copy of the AdminData with the specified title
func (a *AdminData) WithTitle(title string) *AdminData {
	a.Title = title
	return a
}

// WithSuccess returns a copy of the AdminData with a success message
func (a *AdminData) WithSuccess(msg string) *AdminData {
	a.Success = msg
	return a
}

// WithError returns a copy of the AdminData with an error message
func (a *AdminData) WithError(err string) *AdminData {
	a.Error = err
	return a
}

// WithAuthenticated returns a copy of the AdminData with authentication status
func (a *AdminData) WithAuthenticated(authenticated bool) *AdminData {
	a.Authenticated = authenticated
	return a
}

// WithManufacturers returns a copy of the AdminData with manufacturers
func (a *AdminData) WithManufacturers(manufacturers []models.Manufacturer) *AdminData {
	a.Manufacturers = manufacturers
	return a
}

// WithManufacturer returns a copy of the AdminData with a manufacturer
func (a *AdminData) WithManufacturer(manufacturer *models.Manufacturer) *AdminData {
	a.Manufacturer = manufacturer
	return a
}

// WithCalibers returns a copy of the AdminData with calibers
func (a *AdminData) WithCalibers(calibers []models.Caliber) *AdminData {
	a.Calibers = calibers
	return a
}

// WithCaliber returns a copy of the AdminData with a caliber
func (a *AdminData) WithCaliber(caliber *models.Caliber) *AdminData {
	a.Caliber = caliber
	return a
}

// WithWeaponTypes returns a copy of the AdminData with weapon types
func (a *AdminData) WithWeaponTypes(weaponTypes []models.WeaponType) *AdminData {
	a.WeaponTypes = weaponTypes
	return a
}

// WithWeaponType returns a copy of the AdminData with a weapon type
func (a *AdminData) WithWeaponType(weaponType *models.WeaponType) *AdminData {
	a.WeaponType = weaponType
	return a
}

// WithRoles returns a copy of the AdminData with user roles
func (a *AdminData) WithRoles(roles []string) *AdminData {
	// Call the parent WithRoles
	authData := a.AuthData.WithRoles(roles)
	a.AuthData = authData
	return a
}
