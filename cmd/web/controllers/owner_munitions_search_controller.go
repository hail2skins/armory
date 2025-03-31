package controllers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/owner/munitions"
	"github.com/hail2skins/armory/internal/models"
	"gorm.io/gorm"
)

// SearchBrands handles the search for brands
func SearchBrands(c *gin.Context) {
	query := strings.ToLower(c.PostForm("brand_search"))
	db := c.MustGet("db").(*gorm.DB)

	// Get all brands from the database
	brands, err := models.GetAllBrands(db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch brands"})
		return
	}

	// Filter brands based on the search query
	var filteredBrands []models.Brand
	for _, brand := range brands {
		if strings.Contains(strings.ToLower(brand.Name), query) {
			filteredBrands = append(filteredBrands, brand)
		}
	}

	// Render the results template
	munitions.BrandResults(filteredBrands).Render(c.Request.Context(), c.Writer)
}

// SearchCalibers handles the search for calibers
func SearchCalibers(c *gin.Context) {
	query := strings.ToLower(c.PostForm("caliber_search"))
	db := c.MustGet("db").(*gorm.DB)

	// Get all calibers from the database
	calibers, err := models.GetAllCalibers(db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch calibers"})
		return
	}

	// Filter calibers based on the search query
	var filteredCalibers []models.Caliber
	for _, caliber := range calibers {
		searchText := strings.ToLower(caliber.Caliber)
		if caliber.Nickname != "" {
			searchText += " " + strings.ToLower(caliber.Nickname)
		}
		if strings.Contains(searchText, query) {
			filteredCalibers = append(filteredCalibers, caliber)
		}
	}

	// Render the results template
	munitions.CaliberResults(filteredCalibers).Render(c.Request.Context(), c.Writer)
}

// SearchBulletStyles handles the search for bullet styles
func SearchBulletStyles(c *gin.Context) {
	query := strings.ToLower(c.PostForm("bullet_style_search"))
	db := c.MustGet("db").(*gorm.DB)

	// Get all bullet styles from the database
	styles, err := models.GetAllBulletStyles(db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch bullet styles"})
		return
	}

	// Filter styles based on the search query
	var filteredStyles []models.BulletStyle
	for _, style := range styles {
		if strings.Contains(strings.ToLower(style.Type), query) {
			filteredStyles = append(filteredStyles, style)
		}
	}

	// Render the results template
	munitions.BulletStyleResults(filteredStyles).Render(c.Request.Context(), c.Writer)
}

// SearchGrains handles the search for grain weights
func SearchGrains(c *gin.Context) {
	query := strings.ToLower(c.PostForm("grain_search"))
	db := c.MustGet("db").(*gorm.DB)

	// Get all grains from the database
	grains, err := models.GetAllGrains(db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch grain weights"})
		return
	}

	// Filter grains based on the search query
	var filteredGrains []models.Grain
	for _, grain := range grains {
		if strings.Contains(strings.ToLower(fmt.Sprintf("%d", grain.Weight)), query) {
			filteredGrains = append(filteredGrains, grain)
		}
	}

	// Render the results template
	munitions.GrainResults(filteredGrains).Render(c.Request.Context(), c.Writer)
}

// SearchCasings handles the search for casing types
func SearchCasings(c *gin.Context) {
	query := strings.ToLower(c.PostForm("casing_search"))
	db := c.MustGet("db").(*gorm.DB)

	// Get all casings from the database
	casings, err := models.GetAllCasings(db)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch casing types"})
		return
	}

	// Filter casings based on the search query
	var filteredCasings []models.Casing
	for _, casing := range casings {
		if strings.Contains(strings.ToLower(casing.Type), query) {
			filteredCasings = append(filteredCasings, casing)
		}
	}

	// Render the results template
	munitions.CasingResults(filteredCasings).Render(c.Request.Context(), c.Writer)
}
