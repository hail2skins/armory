package tests

import (
	"testing"
	"time"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RangeDayModelTestSuite tests the RangeDay model CRUD functions.
type RangeDayModelTestSuite struct {
	suite.Suite
	DB *gorm.DB
}

func (s *RangeDayModelTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	err = db.AutoMigrate(
		&models.Range{},
		&models.WeaponType{},
		&models.Caliber{},
		&models.Manufacturer{},
		&models.Gun{},
		&models.Brand{},
		&models.BulletStyle{},
		&models.Grain{},
		&models.Casing{},
		&models.Ammo{},
		&models.RangeDay{},
	)
	s.Require().NoError(err)

	s.DB = db
}

func (s *RangeDayModelTestSuite) seedRangeDayDependencies(userID uint) (models.Range, models.Gun, models.Ammo) {
	r := models.Range{RangeName: "Training Grounds", City: "Austin", State: "TX", Zip: "78701"}
	s.Require().NoError(s.DB.Create(&r).Error)

	wt := models.WeaponType{Type: "Pistol"}
	cal := models.Caliber{Caliber: "9mm"}
	man := models.Manufacturer{Name: "Glock", Country: "Austria"}
	s.Require().NoError(s.DB.Create(&wt).Error)
	s.Require().NoError(s.DB.Create(&cal).Error)
	s.Require().NoError(s.DB.Create(&man).Error)

	gun := models.Gun{
		Name:           "G19",
		SerialNumber:   "SN-123",
		WeaponTypeID:   wt.ID,
		CaliberID:      cal.ID,
		ManufacturerID: man.ID,
		OwnerID:        userID,
	}
	s.Require().NoError(s.DB.Create(&gun).Error)

	brand := models.Brand{Name: "Federal"}
	bs := models.BulletStyle{Type: "FMJ"}
	grain := models.Grain{Weight: 115}
	casing := models.Casing{Type: "Brass"}
	s.Require().NoError(s.DB.Create(&brand).Error)
	s.Require().NoError(s.DB.Create(&bs).Error)
	s.Require().NoError(s.DB.Create(&grain).Error)
	s.Require().NoError(s.DB.Create(&casing).Error)

	ammo := models.Ammo{
		Name:          "9mm FMJ",
		BrandID:       brand.ID,
		BulletStyleID: bs.ID,
		GrainID:       grain.ID,
		CaliberID:     cal.ID,
		CasingID:      casing.ID,
		OwnerID:       userID,
		Count:         500,
	}
	s.Require().NoError(s.DB.Create(&ammo).Error)

	return r, gun, ammo
}

func (s *RangeDayModelTestSuite) TestFindRangeDaysByUser_WithExistingRecords_ReturnsUserRecords() {
	userID := uint(1001)
	r, gun, ammo := s.seedRangeDayDependencies(userID)

	rd1 := &models.RangeDay{UserID: userID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Session 1", ShotsFired: 100}
	rd2 := &models.RangeDay{UserID: userID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Session 2", ShotsFired: 120}
	other := &models.RangeDay{UserID: 2002, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Other user", ShotsFired: 50}

	s.Require().NoError(models.CreateRangeDay(s.DB, rd1))
	s.Require().NoError(models.CreateRangeDay(s.DB, rd2))
	s.Require().NoError(models.CreateRangeDay(s.DB, other))

	rangeDays, err := models.FindRangeDaysByUser(s.DB, userID)
	s.Require().NoError(err)
	s.Len(rangeDays, 2)
	for _, rd := range rangeDays {
		s.Equal(userID, rd.UserID)
		s.NotZero(rd.Range.ID)
		s.NotZero(rd.Gun.ID)
		s.NotZero(rd.Ammo.ID)
	}
}

func (s *RangeDayModelTestSuite) TestFindRangeDayByID_WithMatchingUser_ReturnsRecord() {
	userID := uint(3003)
	r, gun, ammo := s.seedRangeDayDependencies(userID)

	rd := &models.RangeDay{UserID: userID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Good practice", ShotsFired: 140}
	s.Require().NoError(models.CreateRangeDay(s.DB, rd))

	found, err := models.FindRangeDayByID(s.DB, rd.ID, userID)
	s.Require().NoError(err)
	s.Equal(rd.ID, found.ID)
	s.Equal("Good practice", found.Comments)
	s.Equal(r.ID, found.Range.ID)
	s.Equal(gun.ID, found.Gun.ID)
	s.Equal(ammo.ID, found.Ammo.ID)
}

func (s *RangeDayModelTestSuite) TestFindRangeDayByID_WithWrongUser_ReturnsError() {
	ownerID := uint(4004)
	r, gun, ammo := s.seedRangeDayDependencies(ownerID)

	rd := &models.RangeDay{UserID: ownerID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Private record", ShotsFired: 80}
	s.Require().NoError(models.CreateRangeDay(s.DB, rd))

	found, err := models.FindRangeDayByID(s.DB, rd.ID, 9999)
	s.Error(err)
	s.Nil(found)
}

func (s *RangeDayModelTestSuite) TestCreateRangeDay_WithValidData_PersistsRecord() {
	userID := uint(5005)
	r, gun, ammo := s.seedRangeDayDependencies(userID)

	rd := &models.RangeDay{UserID: userID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Initial session", ShotsFired: 90}
	err := models.CreateRangeDay(s.DB, rd)
	s.Require().NoError(err)
	s.NotZero(rd.ID)

	stored, err := models.FindRangeDayByID(s.DB, rd.ID, userID)
	s.Require().NoError(err)
	s.Equal("Initial session", stored.Comments)
	s.Equal(90, stored.ShotsFired)
}

func (s *RangeDayModelTestSuite) TestUpdateRangeDay_WithExistingRecord_UpdatesFields() {
	userID := uint(6006)
	r, gun, ammo := s.seedRangeDayDependencies(userID)

	rd := &models.RangeDay{UserID: userID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Before update", ShotsFired: 60}
	s.Require().NoError(models.CreateRangeDay(s.DB, rd))

	rd.Comments = "After update"
	rd.ShotsFired = 200
	rd.Date = time.Now().Add(24 * time.Hour)

	err := models.UpdateRangeDay(s.DB, rd)
	s.Require().NoError(err)

	updated, err := models.FindRangeDayByID(s.DB, rd.ID, userID)
	s.Require().NoError(err)
	s.Equal("After update", updated.Comments)
	s.Equal(200, updated.ShotsFired)
}

func (s *RangeDayModelTestSuite) TestDeleteRangeDay_WithOwnerUser_DeletesRecord() {
	userID := uint(7007)
	r, gun, ammo := s.seedRangeDayDependencies(userID)

	rd := &models.RangeDay{UserID: userID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "To delete", ShotsFired: 110}
	s.Require().NoError(models.CreateRangeDay(s.DB, rd))

	err := models.DeleteRangeDay(s.DB, rd.ID, userID)
	s.Require().NoError(err)

	_, err = models.FindRangeDayByID(s.DB, rd.ID, userID)
	s.Error(err)
}

func (s *RangeDayModelTestSuite) TestDeleteRangeDay_WithNonOwnerUser_ReturnsAuthorizationError() {
	ownerID := uint(8008)
	r, gun, ammo := s.seedRangeDayDependencies(ownerID)

	rd := &models.RangeDay{UserID: ownerID, RangeID: r.ID, GunID: gun.ID, AmmoID: ammo.ID, Date: time.Now(), Comments: "Protected", ShotsFired: 75}
	s.Require().NoError(models.CreateRangeDay(s.DB, rd))

	err := models.DeleteRangeDay(s.DB, rd.ID, 1234)
	s.Error(err)
	s.Contains(err.Error(), "not authorized")
}

func TestRangeDayModelSuite(t *testing.T) {
	suite.Run(t, new(RangeDayModelTestSuite))
}
