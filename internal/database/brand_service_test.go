package database

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// BrandServiceTestSuite defines the test suite for Brand service methods
type BrandServiceTestSuite struct {
	suite.Suite
	db        *gorm.DB
	service   Service
	testBrand *models.Brand
	tempDir   string
}

// SetupTest runs before each test in the suite
func (s *BrandServiceTestSuite) SetupTest() {
	// Create a test database
	db, tempDir := setupTestDB(s.T())
	s.db = db
	s.tempDir = tempDir

	// Run migrations to ensure brand table exists
	err := s.db.AutoMigrate(&models.Brand{})
	s.Require().NoError(err)

	// Initialize the service
	s.service = &service{db: s.db}

	// Create a test brand
	s.testBrand = &models.Brand{
		Name:       "Test Brand",
		Nickname:   "TB",
		Popularity: 100,
	}

	// Save the test brand
	err = s.db.Create(s.testBrand).Error
	s.Require().NoError(err)
}

// TearDownTest runs after each test in the suite
func (s *BrandServiceTestSuite) TearDownTest() {
	// Clean up - delete the test brand
	err := s.db.Unscoped().Delete(&models.Brand{}, s.testBrand.ID).Error
	s.Require().NoError(err)

	// Close the database connection
	sqlDB, err := s.db.DB()
	s.Require().NoError(err)
	err = sqlDB.Close()
	s.Require().NoError(err)
}

// TestFindAllBrands tests the FindAllBrands method
func (s *BrandServiceTestSuite) TestFindAllBrands() {
	// Call the method to test
	brands, err := s.service.FindAllBrands()

	// Assert that there was no error and brands were found
	s.Require().NoError(err)
	s.Require().NotEmpty(brands)

	// Find our test brand in the results
	found := false
	for _, brand := range brands {
		if brand.ID == s.testBrand.ID {
			found = true
			s.Equal("Test Brand", brand.Name)
			s.Equal("TB", brand.Nickname)
			s.Equal(100, brand.Popularity)
			break
		}
	}
	s.True(found, "Test brand should be found in the results")
}

// TestFindBrandByID tests the FindBrandByID method
func (s *BrandServiceTestSuite) TestFindBrandByID() {
	// Call the method to test
	brand, err := s.service.FindBrandByID(s.testBrand.ID)

	// Assert that there was no error and the brand was found
	s.Require().NoError(err)
	s.Require().NotNil(brand)
	s.Equal(s.testBrand.ID, brand.ID)
	s.Equal("Test Brand", brand.Name)
	s.Equal("TB", brand.Nickname)
	s.Equal(100, brand.Popularity)
}

// TestCreateBrand tests the CreateBrand method
func (s *BrandServiceTestSuite) TestCreateBrand() {
	// Create a new brand
	newBrand := &models.Brand{
		Name:       "New Test Brand",
		Nickname:   "NTB",
		Popularity: 50,
	}

	// Call the method to test
	err := s.service.CreateBrand(newBrand)

	// Assert that there was no error and the brand was created
	s.Require().NoError(err)
	s.Require().NotZero(newBrand.ID)

	// Verify the brand was created with correct values
	createdBrand, err := s.service.FindBrandByID(newBrand.ID)
	s.Require().NoError(err)
	s.Equal("New Test Brand", createdBrand.Name)
	s.Equal("NTB", createdBrand.Nickname)
	s.Equal(50, createdBrand.Popularity)

	// Clean up - delete the created brand
	err = s.db.Unscoped().Delete(&models.Brand{}, newBrand.ID).Error
	s.Require().NoError(err)
}

// TestUpdateBrand tests the UpdateBrand method
func (s *BrandServiceTestSuite) TestUpdateBrand() {
	// First, get the brand to update
	brand, err := s.service.FindBrandByID(s.testBrand.ID)
	s.Require().NoError(err)

	// Update the brand
	brand.Name = "Updated Brand"
	brand.Nickname = "UB"
	brand.Popularity = 200

	// Call the method to test
	err = s.service.UpdateBrand(brand)
	s.Require().NoError(err)

	// Verify the brand was updated with correct values
	updatedBrand, err := s.service.FindBrandByID(s.testBrand.ID)
	s.Require().NoError(err)
	s.Equal("Updated Brand", updatedBrand.Name)
	s.Equal("UB", updatedBrand.Nickname)
	s.Equal(200, updatedBrand.Popularity)
}

// TestDeleteBrand tests the DeleteBrand method
func (s *BrandServiceTestSuite) TestDeleteBrand() {
	// Create a temporary brand to delete
	tempBrand := &models.Brand{
		Name:       "Temp Brand",
		Nickname:   "TB",
		Popularity: 10,
	}
	err := s.db.Create(tempBrand).Error
	s.Require().NoError(err)
	s.Require().NotZero(tempBrand.ID)

	// Call the method to test
	err = s.service.DeleteBrand(tempBrand.ID)
	s.Require().NoError(err)

	// Verify the brand was deleted
	_, err = s.service.FindBrandByID(tempBrand.ID)
	s.Require().Error(err)
}

// TestBrandServiceSuite runs the test suite
func TestBrandServiceSuite(t *testing.T) {
	suite.Run(t, new(BrandServiceTestSuite))
}
