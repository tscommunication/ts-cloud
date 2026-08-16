package services

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLocationCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(":memory:"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Division{},
		&models.District{},
		&models.Upazila{},
		&models.PostOffice{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func TestSyncApprovedLocationCatalogIsHierarchicalAndIdempotent(
	t *testing.T,
) {
	db := setupLocationCatalogTestDB(t)

	entries := []approvedLocationCatalogEntry{
		{
			Division:   " Khulna ",
			District:   "Magura",
			Upazila:    "Magura   Sadar",
			PostOffice: "Magura",
			PostalCode: "7600",
		},
		{
			Division: "Khulna",
			District: "Magura",
			Upazila:  "Sreepur",
		},
	}

	first, err := syncApprovedLocationCatalog(db, entries)
	if err != nil {
		t.Fatal(err)
	}

	if first.CreatedDivisions != 1 ||
		first.CreatedDistricts != 1 ||
		first.CreatedUpazilas != 2 ||
		first.CreatedPostOffices != 1 {
		t.Fatalf("unexpected first sync result: %+v", first)
	}

	second, err := syncApprovedLocationCatalog(db, entries)
	if err != nil {
		t.Fatal(err)
	}

	if second.CreatedDivisions != 0 ||
		second.CreatedDistricts != 0 ||
		second.CreatedUpazilas != 0 ||
		second.CreatedPostOffices != 0 {
		t.Fatalf("sync is not idempotent: %+v", second)
	}

	var divisions int64
	var districts int64
	var upazilas int64
	var postOffices int64

	db.Model(&models.Division{}).Count(&divisions)
	db.Model(&models.District{}).Count(&districts)
	db.Model(&models.Upazila{}).Count(&upazilas)
	db.Model(&models.PostOffice{}).Count(&postOffices)

	if divisions != 1 ||
		districts != 1 ||
		upazilas != 2 ||
		postOffices != 1 {
		t.Fatalf(
			"unexpected hierarchy counts: divisions=%d districts=%d upazilas=%d post_offices=%d",
			divisions,
			districts,
			upazilas,
			postOffices,
		)
	}
}

func TestLocationCatalogPostalCodeIsOptionalAndRefreshable(
	t *testing.T,
) {
	db := setupLocationCatalogTestDB(t)

	entry := []approvedLocationCatalogEntry{
		{
			Division:   "Dhaka",
			District:   "Dhaka",
			Upazila:    "Dhamrai",
			PostOffice: "Dhamrai",
		},
	}

	if _, err := syncApprovedLocationCatalog(db, entry); err != nil {
		t.Fatal(err)
	}

	var postOffice models.PostOffice
	if err := db.First(&postOffice).Error; err != nil {
		t.Fatal(err)
	}

	if postOffice.PostalCode != "" {
		t.Fatalf(
			"expected optional postal code to remain empty, got %q",
			postOffice.PostalCode,
		)
	}

	entry[0].PostalCode = "1350"

	if _, err := syncApprovedLocationCatalog(db, entry); err != nil {
		t.Fatal(err)
	}

	if err := db.First(&postOffice, postOffice.ID).Error; err != nil {
		t.Fatal(err)
	}

	if postOffice.PostalCode != "1350" {
		t.Fatalf(
			"expected refreshed postal code 1350, got %q",
			postOffice.PostalCode,
		)
	}
}

func TestLocationCatalogRejectsIncompleteHierarchy(t *testing.T) {
	db := setupLocationCatalogTestDB(t)

	_, err := syncApprovedLocationCatalog(
		db,
		[]approvedLocationCatalogEntry{
			{
				Division: "Dhaka",
				District: "Dhaka",
			},
		},
	)

	if err == nil {
		t.Fatal("expected incomplete hierarchy to be rejected")
	}
}
