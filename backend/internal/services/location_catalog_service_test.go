package services

import (
	"strings"
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

func TestParseApprovedLocationCatalog(t *testing.T) {
	input := strings.NewReader(
		"division,district,upazila,post_office,postal_code\n" +
			" Khulna , Magura ,Magura   Sadar, Magura ,7600\n" +
			"Khulna,Magura,Sreepur,,\n",
	)

	entries, err := parseApprovedLocationCatalog(input)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf(
			"expected 2 catalog entries, got %d",
			len(entries),
		)
	}

	if entries[0].Division != "Khulna" ||
		entries[0].District != "Magura" ||
		entries[0].Upazila != "Magura Sadar" ||
		entries[0].PostOffice != "Magura" ||
		entries[0].PostalCode != "7600" {
		t.Fatalf(
			"unexpected first catalog entry: %+v",
			entries[0],
		)
	}

	if entries[1].Division != "Khulna" ||
		entries[1].District != "Magura" ||
		entries[1].Upazila != "Sreepur" ||
		entries[1].PostOffice != "" ||
		entries[1].PostalCode != "" {
		t.Fatalf(
			"unexpected second catalog entry: %+v",
			entries[1],
		)
	}
}

func TestParseApprovedLocationCatalogRejectsBadHeader(t *testing.T) {
	input := strings.NewReader(
		"division,district,upazila,postal_code\n" +
			"Dhaka,Dhaka,Dhamrai,1350\n",
	)

	if _, err := parseApprovedLocationCatalog(input); err == nil {
		t.Fatal("expected bad catalog header to be rejected")
	}
}

func TestParseApprovedLocationCatalogRejectsIncompleteHierarchy(
	t *testing.T,
) {
	input := strings.NewReader(
		"division,district,upazila,post_office,postal_code\n" +
			"Dhaka,Dhaka,,Dhamrai,1350\n",
	)

	if _, err := parseApprovedLocationCatalog(input); err == nil {
		t.Fatal("expected incomplete hierarchy to be rejected")
	}
}

func TestEmbeddedApprovedLocationCatalogContract(t *testing.T) {
	entries, err := approvedLocationCatalog()
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2496 {
		t.Fatalf(
			"expected 2496 contract catalog entries, got %d",
			len(entries),
		)
	}

	divisions := make(map[string]struct{})
	districtPaths := make(map[string]struct{})
	upazilaPaths := make(map[string]struct{})
	postOfficePaths := make(map[string]struct{})
	fallbackPaths := make(map[string]struct{})
	postalPaths := make(map[string]struct{})

	postalRows := 0
	codedPostalRows := 0
	blankCodePostalRows := 0
	fallbackRows := 0

	for i, entry := range entries {
		if entry.Division == "" ||
			entry.District == "" ||
			entry.Upazila == "" {
			t.Fatalf(
				"catalog entry %d has incomplete hierarchy: %#v",
				i+1,
				entry,
			)
		}

		divisions[entry.Division] = struct{}{}

		districtKey :=
			entry.Division +
				"\x00" +
				entry.District

		districtPaths[districtKey] = struct{}{}

		upazilaKey :=
			districtKey +
				"\x00" +
				entry.Upazila

		upazilaPaths[upazilaKey] = struct{}{}

		if entry.PostOffice == "" {
			if entry.PostalCode != "" {
				t.Fatalf(
					"catalog entry %d has postal code %q without post office",
					i+1,
					entry.PostalCode,
				)
			}

			if _, exists := fallbackPaths[upazilaKey]; exists {
				t.Fatalf(
					"duplicate hierarchy fallback row: %s / %s / %s",
					entry.Division,
					entry.District,
					entry.Upazila,
				)
			}

			fallbackPaths[upazilaKey] = struct{}{}
			fallbackRows++
			continue
		}

		postalRows++
		postalPaths[upazilaKey] = struct{}{}

		postOfficeKey :=
			upazilaKey +
				"\x00" +
				strings.ToLower(entry.PostOffice)

		if _, exists := postOfficePaths[postOfficeKey]; exists {
			t.Fatalf(
				"duplicate post office identity: %s / %s / %s / %s",
				entry.Division,
				entry.District,
				entry.Upazila,
				entry.PostOffice,
			)
		}

		postOfficePaths[postOfficeKey] = struct{}{}

		if entry.PostalCode == "" {
			blankCodePostalRows++
		} else {
			codedPostalRows++
		}
	}

	if len(divisions) != 8 {
		t.Fatalf(
			"expected 8 divisions, got %d",
			len(divisions),
		)
	}

	if len(districtPaths) != 64 {
		t.Fatalf(
			"expected 64 district paths, got %d",
			len(districtPaths),
		)
	}

	if len(upazilaPaths) != 622 {
		t.Fatalf(
			"expected 622 unique upazila paths, got %d",
			len(upazilaPaths),
		)
	}

	if postalRows != 2338 {
		t.Fatalf(
			"expected 2338 postal rows, got %d",
			postalRows,
		)
	}

	if codedPostalRows != 1279 {
		t.Fatalf(
			"expected 1279 coded postal rows, got %d",
			codedPostalRows,
		)
	}

	if blankCodePostalRows != 1059 {
		t.Fatalf(
			"expected 1059 blank-code postal rows, got %d",
			blankCodePostalRows,
		)
	}

	if fallbackRows != 158 {
		t.Fatalf(
			"expected 158 fallback rows, got %d",
			fallbackRows,
		)
	}

	if len(postOfficePaths) != 2338 {
		t.Fatalf(
			"expected 2338 unique post office identities, got %d",
			len(postOfficePaths),
		)
	}

	for upazilaKey := range fallbackPaths {
		if _, exists := postalPaths[upazilaKey]; exists {
			t.Fatalf(
				"upazila has both postal rows and fallback row: %q",
				upazilaKey,
			)
		}
	}

	if len(postalPaths)+len(fallbackPaths) != 622 {
		t.Fatalf(
			"expected 622 postal-covered or fallback upazilas, got %d",
			len(postalPaths)+len(fallbackPaths),
		)
	}
}
