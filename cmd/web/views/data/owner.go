package data

import (
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// UserViewModel represents user data for display in views
type UserViewModel struct {
	Email              string
	CreatedAt          time.Time
	SubscriptionTier   string
	SubscriptionStatus string
}

// OwnerData contains data for owner views
type OwnerData struct {
	// Auth information
	Auth AuthData

	// User information
	User *UserViewModel

	// For guns
	Guns []models.Gun
	Gun  *models.Gun

	// For gun form
	WeaponTypes   []models.WeaponType
	Calibers      []models.Caliber
	Manufacturers []models.Manufacturer
	FormErrors    map[string]string

	// For subscription information
	HasActiveSubscription bool
	SubscriptionTier      string
	SubscriptionEndsAt    string

	// For payment history
	Payments []models.Payment
}

// NewOwnerData creates a new OwnerData with default values
func NewOwnerData() *OwnerData {
	return &OwnerData{
		Auth:       NewAuthData(),
		FormErrors: make(map[string]string),
	}
}

// WithTitle returns a copy of the OwnerData with the specified title
func (o *OwnerData) WithTitle(title string) *OwnerData {
	o.Auth.Title = title
	return o
}

// WithSuccess returns a copy of the OwnerData with a success message
func (o *OwnerData) WithSuccess(msg string) *OwnerData {
	o.Auth.Success = msg
	return o
}

// WithError returns a copy of the OwnerData with an error message
func (o *OwnerData) WithError(err string) *OwnerData {
	o.Auth.Error = err
	return o
}

// WithAuthenticated returns a copy of the OwnerData with authentication status
func (o *OwnerData) WithAuthenticated(authenticated bool) *OwnerData {
	o.Auth.Authenticated = authenticated
	return o
}

// WithUser returns a copy of the OwnerData with user information
func (o *OwnerData) WithUser(dbUser *database.User) *OwnerData {
	o.User = &UserViewModel{
		Email:              dbUser.Email,
		CreatedAt:          dbUser.CreatedAt,
		SubscriptionTier:   dbUser.SubscriptionTier,
		SubscriptionStatus: dbUser.SubscriptionStatus,
	}
	o.Auth.Email = dbUser.Email
	return o
}

// WithGuns returns a copy of the OwnerData with guns
func (o *OwnerData) WithGuns(guns []models.Gun) *OwnerData {
	o.Guns = guns
	return o
}

// WithGun returns a copy of the OwnerData with a gun
func (o *OwnerData) WithGun(gun *models.Gun) *OwnerData {
	o.Gun = gun
	return o
}

// WithWeaponTypes returns a copy of the OwnerData with weapon types
func (o *OwnerData) WithWeaponTypes(weaponTypes []models.WeaponType) *OwnerData {
	o.WeaponTypes = weaponTypes
	return o
}

// WithCalibers returns a copy of the OwnerData with calibers
func (o *OwnerData) WithCalibers(calibers []models.Caliber) *OwnerData {
	o.Calibers = calibers
	return o
}

// WithManufacturers returns a copy of the OwnerData with manufacturers
func (o *OwnerData) WithManufacturers(manufacturers []models.Manufacturer) *OwnerData {
	o.Manufacturers = manufacturers
	return o
}

// WithFormErrors returns a copy of the OwnerData with form errors
func (o *OwnerData) WithFormErrors(errors map[string]string) *OwnerData {
	o.FormErrors = errors
	return o
}

// WithSubscriptionInfo returns a copy of the OwnerData with subscription information
func (o *OwnerData) WithSubscriptionInfo(hasActiveSubscription bool, tier string, endsAt string) *OwnerData {
	o.HasActiveSubscription = hasActiveSubscription
	o.SubscriptionTier = tier
	o.SubscriptionEndsAt = endsAt
	return o
}

// WithPayments returns a copy of the OwnerData with payment history
func (o *OwnerData) WithPayments(payments []models.Payment) *OwnerData {
	o.Payments = payments
	return o
}
