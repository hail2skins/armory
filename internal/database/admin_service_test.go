package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hail2skins/armory/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Helper function to set up the test database and service for admin tests
func setupAdminTestDB(t *testing.T) (*service, func()) {
	// Create a temporary directory for the SQLite database
	tempDir, err := os.MkdirTemp("", "admin-test-*")
	require.NoError(t, err)

	// Create a SQLite database in the temporary directory
	dbPath := filepath.Join(tempDir, "admin-test.db")

	// Open connection to database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Keep logs silent for tests
	})
	require.NoError(t, err)
	require.NotNil(t, db, "Database connection should not be nil")

	// Run migrations for models used in admin service
	// Note: Using &User{} because User is defined in this package
	err = db.AutoMigrate(&models.Manufacturer{}, &models.Caliber{}, &models.WeaponType{}, &User{})
	require.NoError(t, err, "Failed to migrate admin models")

	// Instantiate the service struct directly, injecting the test db
	svc := &service{db: db}
	require.NotNil(t, svc, "Service should not be nil")

	// Define cleanup function
	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
		os.RemoveAll(tempDir)
	}

	// Return the service and the cleanup function
	return svc, cleanup
}

func TestAdminService_ManufacturerCRUD(t *testing.T) {
	svc, cleanup := setupAdminTestDB(t)
	defer cleanup()

	// 1. Create
	m1 := models.Manufacturer{Name: "TestManu1"}
	err := svc.CreateManufacturer(&m1)
	require.NoError(t, err, "CreateManufacturer should not return an error")
	require.NotZero(t, m1.ID, "Manufacturer ID should be populated after creation")

	// 2. FindByID (Verify Create)
	retrievedM1, err := svc.FindManufacturerByID(m1.ID)
	require.NoError(t, err, "FindManufacturerByID should not return an error for created manufacturer")
	require.NotNil(t, retrievedM1, "Retrieved manufacturer should not be nil")
	assert.Equal(t, m1.Name, retrievedM1.Name, "Retrieved manufacturer name should match created name")

	// 3. FindAll (Verify Create)
	allM, err := svc.FindAllManufacturers()
	require.NoError(t, err, "FindAllManufacturers should not return an error")
	assert.Len(t, allM, 1, "FindAllManufacturers should return 1 manufacturer")
	assert.Equal(t, m1.Name, allM[0].Name, "FindAllManufacturers should contain the created manufacturer")

	// 4. Update
	retrievedM1.Name = "UpdatedTestManu1"
	err = svc.UpdateManufacturer(retrievedM1)
	require.NoError(t, err, "UpdateManufacturer should not return an error")

	// 5. FindByID (Verify Update)
	updatedM1, err := svc.FindManufacturerByID(m1.ID)
	require.NoError(t, err, "FindManufacturerByID should not return an error for updated manufacturer")
	require.NotNil(t, updatedM1, "Updated manufacturer should not be nil")
	assert.Equal(t, "UpdatedTestManu1", updatedM1.Name, "Retrieved manufacturer name should match updated name")

	// 6. Delete
	err = svc.DeleteManufacturer(m1.ID)
	require.NoError(t, err, "DeleteManufacturer should not return an error")

	// 7. FindByID (Verify Delete)
	deletedM1, err := svc.FindManufacturerByID(m1.ID)
	assert.Error(t, err, "FindManufacturerByID should return an error for deleted manufacturer")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Error should be gorm.ErrRecordNotFound")
	assert.Nil(t, deletedM1, "Deleted manufacturer should be nil")

	// 8. FindAll (Verify Delete)
	allMAfterDelete, err := svc.FindAllManufacturers()
	require.NoError(t, err, "FindAllManufacturers should not return an error after delete")
	assert.Empty(t, allMAfterDelete, "FindAllManufacturers should return empty list after delete")
}

func TestAdminService_CaliberCRUD(t *testing.T) {
	svc, cleanup := setupAdminTestDB(t)
	defer cleanup()

	// 1. Create
	c1 := models.Caliber{Caliber: "9mm"} // Use Caliber field
	err := svc.CreateCaliber(&c1)
	require.NoError(t, err, "CreateCaliber should not return an error")
	require.NotZero(t, c1.ID, "Caliber ID should be populated after creation")

	// 2. FindByID (Verify Create)
	retrievedC1, err := svc.FindCaliberByID(c1.ID)
	require.NoError(t, err, "FindCaliberByID should not return an error for created caliber")
	require.NotNil(t, retrievedC1, "Retrieved caliber should not be nil")
	assert.Equal(t, c1.Caliber, retrievedC1.Caliber, "Retrieved caliber value should match created value")

	// 3. FindAll (Verify Create)
	allC, err := svc.FindAllCalibers()
	require.NoError(t, err, "FindAllCalibers should not return an error")
	assert.Len(t, allC, 1, "FindAllCalibers should return 1 caliber")
	assert.Equal(t, c1.Caliber, allC[0].Caliber, "FindAllCalibers should contain the created caliber")

	// 4. Update
	retrievedC1.Caliber = "45 ACP"
	err = svc.UpdateCaliber(retrievedC1)
	require.NoError(t, err, "UpdateCaliber should not return an error")

	// 5. FindByID (Verify Update)
	updatedC1, err := svc.FindCaliberByID(c1.ID)
	require.NoError(t, err, "FindCaliberByID should not return an error for updated caliber")
	require.NotNil(t, updatedC1, "Updated caliber should not be nil")
	assert.Equal(t, "45 ACP", updatedC1.Caliber, "Retrieved caliber value should match updated value")

	// 6. Delete
	err = svc.DeleteCaliber(c1.ID)
	require.NoError(t, err, "DeleteCaliber should not return an error")

	// 7. FindByID (Verify Delete)
	deletedC1, err := svc.FindCaliberByID(c1.ID)
	assert.Error(t, err, "FindCaliberByID should return an error for deleted caliber")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Error should be gorm.ErrRecordNotFound")
	assert.Nil(t, deletedC1, "Deleted caliber should be nil")

	// 8. FindAll (Verify Delete)
	allCAfterDelete, err := svc.FindAllCalibers()
	require.NoError(t, err, "FindAllCalibers should not return an error after delete")
	assert.Empty(t, allCAfterDelete, "FindAllCalibers should return empty list after delete")
}

func TestAdminService_WeaponTypeCRUD(t *testing.T) {
	svc, cleanup := setupAdminTestDB(t)
	defer cleanup()

	// 1. Create
	wt1 := models.WeaponType{Type: "Pistol"} // Use Type field
	err := svc.CreateWeaponType(&wt1)
	require.NoError(t, err, "CreateWeaponType should not return an error")
	require.NotZero(t, wt1.ID, "WeaponType ID should be populated after creation")

	// 2. FindByID (Verify Create)
	retrievedWt1, err := svc.FindWeaponTypeByID(wt1.ID)
	require.NoError(t, err, "FindWeaponTypeByID should not return an error for created weapon type")
	require.NotNil(t, retrievedWt1, "Retrieved weapon type should not be nil")
	assert.Equal(t, wt1.Type, retrievedWt1.Type, "Retrieved weapon type value should match created value")

	// 3. FindAll (Verify Create)
	allWt, err := svc.FindAllWeaponTypes()
	require.NoError(t, err, "FindAllWeaponTypes should not return an error")
	assert.Len(t, allWt, 1, "FindAllWeaponTypes should return 1 weapon type")
	assert.Equal(t, wt1.Type, allWt[0].Type, "FindAllWeaponTypes should contain the created weapon type")

	// 4. Update
	retrievedWt1.Type = "Rifle"
	err = svc.UpdateWeaponType(retrievedWt1)
	require.NoError(t, err, "UpdateWeaponType should not return an error")

	// 5. FindByID (Verify Update)
	updatedWt1, err := svc.FindWeaponTypeByID(wt1.ID)
	require.NoError(t, err, "FindWeaponTypeByID should not return an error for updated weapon type")
	require.NotNil(t, updatedWt1, "Updated weapon type should not be nil")
	assert.Equal(t, "Rifle", updatedWt1.Type, "Retrieved weapon type value should match updated value")

	// 6. Delete
	err = svc.DeleteWeaponType(wt1.ID)
	require.NoError(t, err, "DeleteWeaponType should not return an error")

	// 7. FindByID (Verify Delete)
	deletedWt1, err := svc.FindWeaponTypeByID(wt1.ID)
	assert.Error(t, err, "FindWeaponTypeByID should return an error for deleted weapon type")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Error should be gorm.ErrRecordNotFound")
	assert.Nil(t, deletedWt1, "Deleted weapon type should be nil")

	// 8. FindAll (Verify Delete)
	allWtAfterDelete, err := svc.FindAllWeaponTypes()
	require.NoError(t, err, "FindAllWeaponTypes should not return an error after delete")
	assert.Empty(t, allWtAfterDelete, "FindAllWeaponTypes should return empty list after delete")
}

// Add tests for Caliber and WeaponType CRUD here...
