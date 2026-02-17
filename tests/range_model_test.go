package tests

import (
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RangeModelTestSuite tests the Range model CRUD functions.
type RangeModelTestSuite struct {
	suite.Suite
	DB *gorm.DB
}

func (s *RangeModelTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	err = db.AutoMigrate(&models.Range{})
	s.Require().NoError(err)

	s.DB = db
}

func (s *RangeModelTestSuite) TestFindAllRanges_WithExistingRanges_ReturnsAllRanges() {
	r1 := &models.Range{RangeName: "North Range", City: "Austin", State: "TX", Zip: "78701"}
	r2 := &models.Range{RangeName: "South Range", City: "Dallas", State: "TX", Zip: "75201"}
	s.Require().NoError(models.CreateRange(s.DB, r1))
	s.Require().NoError(models.CreateRange(s.DB, r2))

	ranges, err := models.FindAllRanges(s.DB)
	s.Require().NoError(err)
	s.Len(ranges, 2)
}

func (s *RangeModelTestSuite) TestFindRangeByID_WithExistingID_ReturnsRange() {
	r := &models.Range{RangeName: "Precision Range", City: "Plano", State: "TX", Zip: "75023"}
	s.Require().NoError(models.CreateRange(s.DB, r))

	found, err := models.FindRangeByID(s.DB, r.ID)
	s.Require().NoError(err)
	s.Equal(r.ID, found.ID)
	s.Equal("Precision Range", found.RangeName)
}

func (s *RangeModelTestSuite) TestFindRangeByID_WithMissingID_ReturnsError() {
	found, err := models.FindRangeByID(s.DB, 99999)
	s.Error(err)
	s.Nil(found)
}

func (s *RangeModelTestSuite) TestCreateRange_WithValidData_PersistsRange() {
	r := &models.Range{
		RangeName:    "Lone Star Range",
		StreetNumber: "123",
		StreetName:   "Main St",
		AddressLine2: "Suite 200",
		City:         "Houston",
		State:        "TX",
		Zip:          "77001",
	}

	err := models.CreateRange(s.DB, r)
	s.Require().NoError(err)
	s.NotZero(r.ID)

	stored, err := models.FindRangeByID(s.DB, r.ID)
	s.Require().NoError(err)
	s.Equal("Lone Star Range", stored.RangeName)
	s.Equal("Houston", stored.City)
}

func (s *RangeModelTestSuite) TestUpdateRange_WithExistingRange_UpdatesFields() {
	r := &models.Range{RangeName: "Old Name", City: "Old City", State: "TX", Zip: "73301"}
	s.Require().NoError(models.CreateRange(s.DB, r))

	r.RangeName = "Updated Name"
	r.City = "Updated City"
	r.StreetNumber = "999"
	r.StreetName = "Updated Blvd"

	err := models.UpdateRange(s.DB, r)
	s.Require().NoError(err)

	updated, err := models.FindRangeByID(s.DB, r.ID)
	s.Require().NoError(err)
	s.Equal("Updated Name", updated.RangeName)
	s.Equal("Updated City", updated.City)
	s.Equal("999", updated.StreetNumber)
}

func (s *RangeModelTestSuite) TestDeleteRange_WithExistingID_SoftDeletesRange() {
	r := &models.Range{RangeName: "Delete Me", City: "Waco", State: "TX", Zip: "76701"}
	s.Require().NoError(models.CreateRange(s.DB, r))

	err := models.DeleteRange(s.DB, r.ID)
	s.Require().NoError(err)

	_, err = models.FindRangeByID(s.DB, r.ID)
	s.Error(err)
}

func (s *RangeModelTestSuite) TestDeleteRange_WithMissingID_ReturnsError() {
	err := models.DeleteRange(s.DB, 424242)
	s.Error(err)
}

func TestRangeModelSuite(t *testing.T) {
	suite.Run(t, new(RangeModelTestSuite))
}
