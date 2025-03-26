package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/partials"
	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestPromotionBanner(t *testing.T) {
	// Create a promotion that applies only to new users
	newUserPromo := &models.Promotion{
		Model:                gorm.Model{ID: 1},
		Name:                 "New User Promo",
		Type:                 "free_trial",
		Active:               true,
		StartDate:            time.Now(),
		EndDate:              time.Now().AddDate(0, 1, 0),
		BenefitDays:          30,
		DisplayOnHome:        true,
		ApplyToExistingUsers: false,
		Description:          "Test promotion for new users",
		Banner:               "/path/to/banner.jpg",
	}

	// Create a promotion that applies to both new and existing users
	existingUserPromo := &models.Promotion{
		Model:                gorm.Model{ID: 2},
		Name:                 "Existing User Promo",
		Type:                 "free_trial",
		Active:               true,
		StartDate:            time.Now(),
		EndDate:              time.Now().AddDate(0, 1, 0),
		BenefitDays:          30,
		DisplayOnHome:        true,
		ApplyToExistingUsers: true,
		Description:          "Test promotion for new and existing users",
		Banner:               "/path/to/banner.jpg",
	}

	// Test cases
	tests := []struct {
		name               string
		promotion          *models.Promotion
		expectedTextPart   string
		unexpectedTextPart string
	}{
		{
			name:               "New user promotion",
			promotion:          newUserPromo,
			expectedTextPart:   "Register now",
			unexpectedTextPart: "OR Login",
		},
		{
			name:               "Existing user promotion",
			promotion:          existingUserPromo,
			expectedTextPart:   "Register OR Login",
			unexpectedTextPart: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create auth data with the promotion
			authData := data.NewAuthData()
			authData.ActivePromotion = tt.promotion

			// Render the banner to a string
			var builder strings.Builder
			err := partials.PromotionBanner(authData).Render(context.Background(), &builder)
			assert.NoError(t, err)

			// Check the output contains the expected text
			result := builder.String()
			assert.Contains(t, result, tt.expectedTextPart)

			// Check it doesn't contain unexpected text
			if tt.unexpectedTextPart != "" {
				assert.NotContains(t, result, tt.unexpectedTextPart)
			}
		})
	}
}
