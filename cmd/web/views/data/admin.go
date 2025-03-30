package data

import (
	"time"

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

	// For casings
	Casings []models.Casing
	Casing  *models.Casing

	// For bullet styles
	BulletStyles []models.BulletStyle
	BulletStyle  *models.BulletStyle

	// For promotions
	Promotions []models.Promotion
	Promotion  *models.Promotion

	// For forms
	FormData map[string]interface{}

	// For dashboard
	TotalUsers                 int64
	UserGrowthRate             float64
	SubscribedUsers            int64
	SubscribedGrowthRate       float64
	NewRegistrations           int64
	NewRegistrationsGrowthRate float64
	NewSubscriptions           int64
	NewSubscriptionsGrowthRate float64
	MonthlySubscribers         int64
	MonthlyGrowthRate          float64
	YearlySubscribers          int64
	YearlyGrowthRate           float64
	LifetimeSubscribers        int64
	LifetimeGrowthRate         float64
	PremiumSubscribers         int64
	PremiumGrowthRate          float64
	RecentUsers                []models.User
	CurrentPage                int
	PerPage                    int
	TotalPages                 int
	SortBy                     string
	SortOrder                  string
	SearchQuery                string

	// For error metrics
	CriticalErrorCount  int64
	WarningCount        int64
	InfoCount           int64
	RecentErrors        []ErrorEntry
	ErrorRatesByService map[string]float64
	TotalErrorRate      float64
	LatencyPercentiles  map[string]float64

	// For system health
	SystemInfo map[string]string
}

// ErrorEntry represents a simplified error record for views
type ErrorEntry struct {
	ErrorType    string
	Count        int64
	LastOccurred time.Time
	Path         string
	Level        string
	Service      string
	Message      string
	IPAddress    string
}

// User interface for dashboard
type User interface {
	GetID() uint
	GetUserName() string
	GetCreatedAt() time.Time
	GetLastLogin() time.Time
	GetSubscriptionTier() string
	IsDeleted() bool
	IsVerified() bool
	GetSubscriptionStatus() string
	GetSubscriptionEndDate() time.Time
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

// WithCasings returns a copy of the AdminData with casings
func (a *AdminData) WithCasings(casings []models.Casing) *AdminData {
	a.Casings = casings
	return a
}

// WithCasing returns a copy of the AdminData with a single casing
func (a *AdminData) WithCasing(casing *models.Casing) *AdminData {
	a.Casing = casing
	return a
}

// WithBulletStyles returns a copy of the AdminData with bullet styles
func (a *AdminData) WithBulletStyles(bulletStyles []models.BulletStyle) *AdminData {
	a.BulletStyles = bulletStyles
	return a
}

// WithBulletStyle returns a copy of the AdminData with a single bullet style
func (a *AdminData) WithBulletStyle(bulletStyle *models.BulletStyle) *AdminData {
	a.BulletStyle = bulletStyle
	return a
}

// WithRoles returns a copy of the AdminData with user roles
func (a *AdminData) WithRoles(roles []string) *AdminData {
	// Call the parent WithRoles
	authData := a.AuthData.WithRoles(roles)
	a.AuthData = authData
	return a
}

// WithDashboardData sets all dashboard statistics data
func (a *AdminData) WithDashboardData(
	totalUsers int64, userGrowthRate float64,
	subscribedUsers int64, subscribedGrowthRate float64,
	newRegistrations int64, newRegistrationsGrowthRate float64,
	newSubscriptions int64, newSubscriptionsGrowthRate float64,
	monthlySubscribers int64, monthlyGrowthRate float64,
	yearlySubscribers int64, yearlyGrowthRate float64,
	lifetimeSubscribers int64, lifetimeGrowthRate float64,
	premiumSubscribers int64, premiumGrowthRate float64,
) *AdminData {
	a.TotalUsers = totalUsers
	a.UserGrowthRate = userGrowthRate
	a.SubscribedUsers = subscribedUsers
	a.SubscribedGrowthRate = subscribedGrowthRate
	a.NewRegistrations = newRegistrations
	a.NewRegistrationsGrowthRate = newRegistrationsGrowthRate
	a.NewSubscriptions = newSubscriptions
	a.NewSubscriptionsGrowthRate = newSubscriptionsGrowthRate
	a.MonthlySubscribers = monthlySubscribers
	a.MonthlyGrowthRate = monthlyGrowthRate
	a.YearlySubscribers = yearlySubscribers
	a.YearlyGrowthRate = yearlyGrowthRate
	a.LifetimeSubscribers = lifetimeSubscribers
	a.LifetimeGrowthRate = lifetimeGrowthRate
	a.PremiumSubscribers = premiumSubscribers
	a.PremiumGrowthRate = premiumGrowthRate
	return a
}

// WithRecentUsers sets the recent users data
func (a *AdminData) WithRecentUsers(users []models.User) *AdminData {
	a.RecentUsers = users
	return a
}

// WithPagination sets the pagination data
func (a *AdminData) WithPagination(currentPage, perPage, totalPages int) *AdminData {
	a.CurrentPage = currentPage
	a.PerPage = perPage
	a.TotalPages = totalPages
	return a
}

// WithSorting sets the sorting data
func (a *AdminData) WithSorting(sortBy, sortOrder string) *AdminData {
	a.SortBy = sortBy
	a.SortOrder = sortOrder
	return a
}

// WithPromotions returns a copy of the AdminData with promotions
func (a *AdminData) WithPromotions(promotions []models.Promotion) *AdminData {
	a.Promotions = promotions
	return a
}

// WithPromotion returns a copy of the AdminData with a promotion
func (a *AdminData) WithPromotion(promotion *models.Promotion) *AdminData {
	a.Promotion = promotion
	return a
}

// WithFormData returns a copy of the AdminData with the specified form data
func (a *AdminData) WithFormData(formData map[string]interface{}) *AdminData {
	a.FormData = formData
	return a
}

// WithErrorMetrics sets the error metrics data
func (a *AdminData) WithErrorMetrics(
	criticalCount, warningCount, infoCount int64,
	recentErrors []ErrorEntry,
	errorRatesByService map[string]float64,
	totalErrorRate float64,
	latencyPercentiles map[string]float64,
) *AdminData {
	a.CriticalErrorCount = criticalCount
	a.WarningCount = warningCount
	a.InfoCount = infoCount
	a.RecentErrors = recentErrors
	a.ErrorRatesByService = errorRatesByService
	a.TotalErrorRate = totalErrorRate
	a.LatencyPercentiles = latencyPercentiles
	return a
}

// WithSystemInfo sets the system information data
func (a *AdminData) WithSystemInfo(systemInfo map[string]string) *AdminData {
	a.SystemInfo = systemInfo
	return a
}
