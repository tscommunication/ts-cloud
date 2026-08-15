package services

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestCustomerExportRowsMapsStructuredAddress(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_export_structured_address?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.POP{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	pop := models.POP{
		Code: "POP-EXPORT",
		Name: "Export POP",
	}
	if err := db.Create(&pop).Error; err != nil {
		t.Fatal(err)
	}

	customer := models.Customer{
		CustomerCode:     "CUS-EXPORT",
		FullName:         "Export Customer",
		Mobile:           "01700000000",
		Country:          "Bangladesh",
		RoadOrArea:       "Area One, Block A, Road 7",
		VillageOrHolding: "Holding 12, Flat 3B",
		Address:          "legacy address must not replace structured address",
		PopID:            &pop.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		Name:   "Export_P5",
		Price:  500,
		Status: "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	subscription := models.Subscription{
		SubscriptionCode: "SUB-EXPORT",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
		ActivationDate:   now,
		ExpiryDate:       now.AddDate(0, 1, 0),
		NextBillingDate:  now.AddDate(0, 1, 0),
		BillingDay:       16,
		PPPoEUsername:    "export-user",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := customerExportRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 export row, got %d", len(rows))
	}

	row := rows[0]
	if len(row) != len(customerExportHeaders) {
		t.Fatalf(
			"expected %d export columns, got %d",
			len(customerExportHeaders),
			len(row),
		)
	}

	index := map[string]int{}
	for i, header := range customerExportHeaders {
		index[header] = i
	}

	if got := row[index["Area"]]; got != customer.RoadOrArea {
		t.Fatalf(
			"expected Area %q, got %q",
			customer.RoadOrArea,
			got,
		)
	}

	if got := row[index["Building Name"]]; got != customer.VillageOrHolding {
		t.Fatalf(
			"expected Building Name %q, got %q",
			customer.VillageOrHolding,
			got,
		)
	}

	for _, header := range []string{
		"Block",
		"Road Name",
		"Road No",
		"Building No",
		"Flat",
	} {
		if got := row[index[header]]; got != "" {
			t.Fatalf("expected %s to be blank, got %q", header, got)
		}
	}
}

func TestCustomerAddressExportImportRoundTrip(t *testing.T) {
	original := models.Customer{
		RoadOrArea:       "Magura Town, Block A, Road 7",
		VillageOrHolding: "Kalinagor House, Holding 12, Flat 3B",
	}

	exportRow := map[string]string{
		"Area":          strings.TrimSpace(original.RoadOrArea),
		"Block":         "",
		"Road Name":     "",
		"Road No":       "",
		"Building Name": strings.TrimSpace(original.VillageOrHolding),
		"Building No":   "",
		"Flat":          "",
	}

	roadOrArea := importAddressParts(
		exportRow,
		"Area",
		"Block",
		"Road Name",
		"Road No",
	)
	villageOrHolding := importAddressParts(
		exportRow,
		"Building Name",
		"Building No",
		"Flat",
	)
	legacyAddress := importAddress(exportRow)

	if roadOrArea != original.RoadOrArea {
		t.Fatalf(
			"RoadOrArea round trip changed: want %q, got %q",
			original.RoadOrArea,
			roadOrArea,
		)
	}

	if villageOrHolding != original.VillageOrHolding {
		t.Fatalf(
			"VillageOrHolding round trip changed: want %q, got %q",
			original.VillageOrHolding,
			villageOrHolding,
		)
	}

	expectedLegacy := original.RoadOrArea + ", " + original.VillageOrHolding
	if legacyAddress != expectedLegacy {
		t.Fatalf(
			"legacy address mismatch: want %q, got %q",
			expectedLegacy,
			legacyAddress,
		)
	}
}

func TestCustomerExportRowsFallsBackToLegacyAddress(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_export_legacy_address?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	customer := models.Customer{
		CustomerCode: "CUS-LEGACY",
		FullName:     "Legacy Customer",
		Mobile:       "01800000000",
		Address:      "Legacy Area",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		Name:   "Legacy_P5",
		Price:  500,
		Status: "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	subscription := models.Subscription{
		SubscriptionCode: "SUB-LEGACY",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
		ActivationDate:   now,
		ExpiryDate:       now.AddDate(0, 1, 0),
		NextBillingDate:  now.AddDate(0, 1, 0),
		BillingDay:       16,
		PPPoEUsername:    "legacy-user",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := customerExportRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 export row, got %d", len(rows))
	}

	index := map[string]int{}
	for i, header := range customerExportHeaders {
		index[header] = i
	}

	if got := rows[0][index["Area"]]; got != "Legacy Area" {
		t.Fatalf("expected legacy Area fallback, got %q", got)
	}

	if got := rows[0][index["Building Name"]]; got != "" {
		t.Fatalf("expected empty Building Name, got %q", got)
	}
}
