package repositories

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLocationRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open("file:location_repository?mode=memory&cache=shared"),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
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

func TestLocationRepositoryHierarchyIsScopedAndSorted(t *testing.T) {
	db := setupLocationRepositoryTestDB(t)

	divisionB := models.Division{Name: "Rajshahi"}
	divisionA := models.Division{Name: "Dhaka"}

	if err := db.Create(&divisionB).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&divisionA).Error; err != nil {
		t.Fatal(err)
	}

	districtB := models.District{
		DivisionID: divisionA.ID,
		Name:       "Gazipur",
	}
	districtA := models.District{
		DivisionID: divisionA.ID,
		Name:       "Dhaka",
	}
	foreignDistrict := models.District{
		DivisionID: divisionB.ID,
		Name:       "Pabna",
	}

	for _, row := range []*models.District{
		&districtB,
		&districtA,
		&foreignDistrict,
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	upazilaB := models.Upazila{
		DistrictID: districtA.ID,
		Name:       "Savar",
	}
	upazilaA := models.Upazila{
		DistrictID: districtA.ID,
		Name:       "Dhamrai",
	}
	foreignUpazila := models.Upazila{
		DistrictID: foreignDistrict.ID,
		Name:       "Ishwardi",
	}

	for _, row := range []*models.Upazila{
		&upazilaB,
		&upazilaA,
		&foreignUpazila,
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	postOfficeB := models.PostOffice{
		UpazilaID:  upazilaA.ID,
		Name:       "Kalampur",
		PostalCode: "1351",
	}
	postOfficeA := models.PostOffice{
		UpazilaID:  upazilaA.ID,
		Name:       "Dhamrai",
		PostalCode: "1350",
	}
	foreignPostOffice := models.PostOffice{
		UpazilaID:  foreignUpazila.ID,
		Name:       "Ishwardi",
		PostalCode: "6620",
	}

	for _, row := range []*models.PostOffice{
		&postOfficeB,
		&postOfficeA,
		&foreignPostOffice,
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	divisions, err := ListDivisions()
	if err != nil {
		t.Fatal(err)
	}
	if len(divisions) != 2 {
		t.Fatalf("expected 2 divisions, got %d", len(divisions))
	}
	if divisions[0].Name != "Dhaka" ||
		divisions[1].Name != "Rajshahi" {
		t.Fatalf("unexpected division order: %+v", divisions)
	}

	districts, err := ListDistrictsByDivision(divisionA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(districts) != 2 {
		t.Fatalf(
			"expected 2 districts for division %d, got %d",
			divisionA.ID,
			len(districts),
		)
	}
	if districts[0].Name != "Dhaka" ||
		districts[1].Name != "Gazipur" {
		t.Fatalf("unexpected district rows: %+v", districts)
	}
	for _, row := range districts {
		if row.DivisionID != divisionA.ID {
			t.Fatalf(
				"foreign district returned: %+v",
				row,
			)
		}
	}

	upazilas, err := ListUpazilasByDistrict(districtA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(upazilas) != 2 {
		t.Fatalf(
			"expected 2 upazilas for district %d, got %d",
			districtA.ID,
			len(upazilas),
		)
	}
	if upazilas[0].Name != "Dhamrai" ||
		upazilas[1].Name != "Savar" {
		t.Fatalf("unexpected upazila rows: %+v", upazilas)
	}
	for _, row := range upazilas {
		if row.DistrictID != districtA.ID {
			t.Fatalf(
				"foreign upazila returned: %+v",
				row,
			)
		}
	}

	postOffices, err := ListPostOfficesByUpazila(upazilaA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(postOffices) != 2 {
		t.Fatalf(
			"expected 2 post offices for upazila %d, got %d",
			upazilaA.ID,
			len(postOffices),
		)
	}
	if postOffices[0].Name != "Dhamrai" ||
		postOffices[1].Name != "Kalampur" {
		t.Fatalf(
			"unexpected post office rows: %+v",
			postOffices,
		)
	}
	if postOffices[0].PostalCode != "1350" ||
		postOffices[1].PostalCode != "1351" {
		t.Fatalf(
			"unexpected postal codes: %+v",
			postOffices,
		)
	}
	for _, row := range postOffices {
		if row.UpazilaID != upazilaA.ID {
			t.Fatalf(
				"foreign post office returned: %+v",
				row,
			)
		}
	}
}

func TestLocationRepositoryUnknownParentReturnsEmptyList(t *testing.T) {
	setupLocationRepositoryTestDB(t)

	districts, err := ListDistrictsByDivision(999999)
	if err != nil {
		t.Fatal(err)
	}
	if len(districts) != 0 {
		t.Fatalf(
			"expected no districts for unknown division, got %+v",
			districts,
		)
	}

	upazilas, err := ListUpazilasByDistrict(999999)
	if err != nil {
		t.Fatal(err)
	}
	if len(upazilas) != 0 {
		t.Fatalf(
			"expected no upazilas for unknown district, got %+v",
			upazilas,
		)
	}

	postOffices, err := ListPostOfficesByUpazila(999999)
	if err != nil {
		t.Fatal(err)
	}
	if len(postOffices) != 0 {
		t.Fatalf(
			"expected no post offices for unknown upazila, got %+v",
			postOffices,
		)
	}
}
