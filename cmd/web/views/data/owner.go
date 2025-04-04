package data

import (
	"strconv"
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

	// For ammunition
	Ammo              []models.Ammo
	AmmoCount         int64
	TotalAmmoQuantity int64
	TotalAmmoPaid     float64
	TotalAmmoExpended int64

	// For gun form
	WeaponTypes   []models.WeaponType
	Calibers      []models.Caliber
	Manufacturers []models.Manufacturer
	FormErrors    map[string]string

	// For ammunition form
	Brands       []models.Brand
	BulletStyles []models.BulletStyle
	Grains       []models.Grain
	Casings      []models.Casing

	// For subscription information
	HasActiveSubscription bool
	SubscriptionTier      string
	SubscriptionEndsAt    string

	// For payment history
	Payments []models.Payment

	// Pagination
	CurrentPage     int
	TotalPages      int
	PerPage         int
	TotalItems      int
	ShowingFrom     int
	ShowingTo       int
	HasNextPage     bool
	HasPreviousPage bool
	StartPage       int
	EndPage         int

	// Sorting and filtering
	SortBy     string
	SortOrder  string
	SearchTerm string

	// For display options
	HasFiltersApplied bool

	// For gun costs
	TotalPaid float64

	// Notes or additional messages to display
	Note string
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

// WithAmmo returns a copy of the OwnerData with ammunition
func (o *OwnerData) WithAmmo(ammo []models.Ammo) *OwnerData {
	o.Ammo = ammo
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

// WithPagination returns a copy of the OwnerData with pagination information
func (o *OwnerData) WithPagination(currentPage, totalPages, perPage, totalItems int) *OwnerData {
	o.CurrentPage = currentPage
	o.TotalPages = totalPages
	o.PerPage = perPage
	o.TotalItems = totalItems

	// Calculate derived pagination values
	o.HasPreviousPage = currentPage > 1
	o.HasNextPage = currentPage < totalPages

	// Calculate showing from/to values
	o.ShowingFrom = (currentPage-1)*perPage + 1
	if totalItems == 0 {
		o.ShowingFrom = 0
	}

	o.ShowingTo = currentPage * perPage
	if o.ShowingTo > totalItems {
		o.ShowingTo = totalItems
	}

	// Calculate page range for pagination links
	o.StartPage = 1
	o.EndPage = totalPages

	// Limit to maximum of 5 pages shown
	if totalPages > 5 {
		// Center around current page
		o.StartPage = currentPage - 2
		o.EndPage = currentPage + 2

		// Adjust if we're near the beginning
		if o.StartPage < 1 {
			o.StartPage = 1
			o.EndPage = 5
		}

		// Adjust if we're near the end
		if o.EndPage > totalPages {
			o.EndPage = totalPages
			o.StartPage = totalPages - 4
			if o.StartPage < 1 {
				o.StartPage = 1
			}
		}
	}

	return o
}

// WithSorting returns a copy of the OwnerData with sorting information
func (o *OwnerData) WithSorting(sortBy, sortOrder string) *OwnerData {
	o.SortBy = sortBy
	o.SortOrder = sortOrder
	return o
}

// WithSearchTerm returns a copy of the OwnerData with search term
func (o *OwnerData) WithSearchTerm(searchTerm string) *OwnerData {
	o.SearchTerm = searchTerm
	return o
}

// WithFiltersApplied returns a copy of the OwnerData indicating if non-default filters are applied
func (o *OwnerData) WithFiltersApplied(sortBy, sortOrder string, perPage int, searchTerm string) *OwnerData {
	// Check if any non-default filters are applied
	o.HasFiltersApplied = (sortBy != "created_at" || sortOrder != "desc" || perPage != 10 || searchTerm != "")
	return o
}

// GetPaginationURL returns a formatted URL with pagination parameters
func (o *OwnerData) GetPaginationURL(page int, path string) string {
	url := path + "?page=" + strconv.Itoa(page) +
		"&perPage=" + strconv.Itoa(o.PerPage) +
		"&sortBy=" + o.SortBy +
		"&sortOrder=" + o.SortOrder

	if o.SearchTerm != "" {
		url += "&search=" + o.SearchTerm
	}

	return url
}

// GetGunURL returns a formatted URL for a gun
func (o *OwnerData) GetGunURL(gun models.Gun, action string) string {
	baseURL := "/owner/guns/" + strconv.FormatUint(uint64(gun.ID), 10)
	if action != "" {
		baseURL += "/" + action
	}
	return baseURL
}

// WithTotalPaid returns a copy of the OwnerData with the total paid amount
func (o *OwnerData) WithTotalPaid(totalPaid float64) *OwnerData {
	o.TotalPaid = totalPaid
	return o
}

// WithAmmoCount returns a copy of the OwnerData with ammunition count
func (o *OwnerData) WithAmmoCount(count int64) *OwnerData {
	o.AmmoCount = count
	return o
}

// WithTotalAmmoQuantity returns a copy of the OwnerData with total ammunition quantity
func (o *OwnerData) WithTotalAmmoQuantity(quantity int64) *OwnerData {
	o.TotalAmmoQuantity = quantity
	return o
}

// WithTotalAmmoPaid returns a copy of the OwnerData with total paid for ammunition
func (o *OwnerData) WithTotalAmmoPaid(totalPaid float64) *OwnerData {
	o.TotalAmmoPaid = totalPaid
	return o
}

// WithTotalAmmoExpended returns a copy of the OwnerData with total expended rounds
func (o *OwnerData) WithTotalAmmoExpended(expended int64) *OwnerData {
	o.TotalAmmoExpended = expended
	return o
}

// WithNote returns a copy of the OwnerData with a note
func (o *OwnerData) WithNote(note string) *OwnerData {
	o.Note = note
	return o
}
