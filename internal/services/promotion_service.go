package services

import (
	"sync"
	"time"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/models"
)

// PromotionService handles promotion-related business logic and provides a caching layer
// for efficient access to active promotions. This service is responsible for:
// - Retrieving active promotions from the database
// - Caching promotions to reduce database load
// - Providing logic to select the "best" promotion for users
// - Managing promotion benefits application
type PromotionService struct {
	DB            database.Service   // Database service for promotion data access
	cacheMutex    sync.RWMutex       // Mutex to ensure thread-safe cache access
	cache         []models.Promotion // In-memory cache of promotions
	cacheExpiry   time.Time          // When the current cache expires
	cacheDuration time.Duration      // How long to keep cached data before refreshing
}

// NewPromotionService creates a new PromotionService with the provided database service
// and default cache settings. The default cache duration is 15 minutes, which provides
// a good balance between performance and data freshness.
func NewPromotionService(db database.Service) *PromotionService {
	return &PromotionService{
		DB:            db,
		cacheDuration: 15 * time.Minute, // 15 minute cache TTL by default
	}
}

// GetActivePromotions returns all currently active promotions from the database
// Active promotions have the Active flag set to true and are within their
// StartDate and EndDate range.
func (s *PromotionService) GetActivePromotions() ([]models.Promotion, error) {
	return s.DB.FindActivePromotions()
}

// GetBestActivePromotion returns the most beneficial promotion that's currently active
// Selection criteria:
// 1. The promotion must be active (Active=true)
// 2. Current time must be between StartDate and EndDate
// 3. Promotions with more BenefitDays are prioritized
// 4. If two promotions have the same BenefitDays, the one ending sooner is chosen to create urgency
//
// Returns nil if no active promotions are found.
func (s *PromotionService) GetBestActivePromotion() (*models.Promotion, error) {
	promotions, err := s.GetActivePromotions()
	if err != nil {
		return nil, err
	}

	if len(promotions) == 0 {
		return nil, nil
	}

	// Find the best promotion according to the criteria
	var bestPromotion *models.Promotion

	now := time.Now()

	for _, promotion := range promotions {
		// Skip promotions that are not active or outside date range
		// This is a double-check since the database query should already filter these
		if !promotion.Active || now.Before(promotion.StartDate) || now.After(promotion.EndDate) {
			continue
		}

		// Use the first valid promotion as the initial best
		if bestPromotion == nil {
			p := promotion // Create a copy to avoid array reference issues
			bestPromotion = &p
			continue
		}

		// Prefer promotions with more benefit days (greater value to user)
		if promotion.BenefitDays > bestPromotion.BenefitDays {
			p := promotion
			bestPromotion = &p
		} else if promotion.BenefitDays == bestPromotion.BenefitDays {
			// If same benefit days, prefer the one ending soonest (creates urgency)
			if promotion.EndDate.Before(bestPromotion.EndDate) {
				p := promotion
				bestPromotion = &p
			}
		}
	}

	return bestPromotion, nil
}

// ClearCache forces the promotion cache to be cleared immediately
// This should be called whenever promotions are created, updated, or deleted
// to ensure users always see the most current promotion data.
func (s *PromotionService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache = nil
	s.cacheExpiry = time.Time{}
}

// getFromCache returns cached promotions if they exist and aren't expired
// This is a private method used internally by the service to optimize
// database access patterns.
// Returns nil if the cache is expired or empty.
func (s *PromotionService) getFromCache() []models.Promotion {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if time.Now().Before(s.cacheExpiry) && len(s.cache) > 0 {
		// Return a copy of the cache to prevent modification
		result := make([]models.Promotion, len(s.cache))
		copy(result, s.cache)
		return result
	}

	return nil
}

// updateCache updates the promotion cache with fresh data
// This is a private method used internally by the service after
// retrieving new data from the database.
func (s *PromotionService) updateCache(promotions []models.Promotion) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Copy the promotions to prevent modification of the cached data
	s.cache = make([]models.Promotion, len(promotions))
	copy(s.cache, promotions)

	// Set the new expiry time based on the configured cache duration
	s.cacheExpiry = time.Now().Add(s.cacheDuration)
}

// SetCacheDuration allows customizing the TTL (time-to-live) of the cache
// This can be useful in testing or in different environments where
// promotion update frequency may vary.
// Setting a longer duration improves performance but may delay updates.
// Setting a shorter duration ensures fresher data but increases database load.
func (s *PromotionService) SetCacheDuration(duration time.Duration) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cacheDuration = duration
	s.cacheExpiry = time.Time{} // Force cache refresh on next access
}
