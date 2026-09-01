package services

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func TestAgentPOPCatalogMatchesApprovedWorkbook(t *testing.T) {
	catalog, err := importAgentPOPCatalogs()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 15 {
		t.Fatalf("expected 15 approved POPs, got %d", len(catalog))
	}

	managers := map[string]bool{}
	hasHeadOffice := false
	for _, item := range catalog {
		managers[normalizedCatalogName(item.ManagerName)] = true
		if normalizedCatalogName(item.POPName) == "delete user" {
			t.Fatal("Delete User must not be present in the approved catalog")
		}
		if item.POPName == "Head Office / Kalinagor" && item.ManagerID == "1" && item.ManagerName == "Md. Tariqul Islam" {
			hasHeadOffice = true
		}
	}
	if len(managers) != 10 {
		t.Fatalf("expected 10 approved managers, got %d", len(managers))
	}
	if !hasHeadOffice {
		t.Fatal("approved Head Office / Kalinagor mapping is missing")
	}
	if normalizedCatalogName("Kasundi &  Bagbaria") != normalizedCatalogName("Kasundi & Bagbaria") {
		t.Fatal("POP normalization must collapse repeated whitespace")
	}
}

func TestPreviewCustomerCSVRejectsUnknownPOP(t *testing.T) {
	csvInput := strings.Join([]string{
		"ID,Username,Status,Package,POP,Name,Contact,Expire,B Cycle",
		"101,user-active,active,Pack:Little_P5,Delete User,Customer,'01700000001',2026-09-14,14",
	}, "\n")
	_, err := PreviewCustomerCSV(strings.NewReader(csvInput))
	if err == nil || !strings.Contains(err.Error(), "missing from the approved Agent/POP catalog") {
		t.Fatalf("expected unknown POP rejection, got %v", err)
	}
}

func TestPreviewCustomerFileReadsXLSX(t *testing.T) {
	book := excelize.NewFile()
	t.Cleanup(func() { _ = book.Close() })
	headers := []interface{}{"ID", "Username", "Status", "Package", "POP", "Name", "Contact", "Expire", "B Cycle"}
	active := []interface{}{"101", "xlsx-active", "active", "Pack:Little_P5", "Kasundi & Bagbaria", "Excel Active", "01700000001", "2026-09-14", "14"}
	inactive := []interface{}{"102", "xlsx-inactive", "deactive", "Pack:Little_P5", "Head Office / Kalinagor", "Excel Inactive", "01700000002", "2026-09-15", "15"}
	for cell, row := range map[string][]interface{}{"A1": headers, "A2": active, "A3": inactive} {
		if err := book.SetSheetRow("Sheet1", cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	var workbook bytes.Buffer
	if err := book.Write(&workbook); err != nil {
		t.Fatal(err)
	}

	preview, err := PreviewCustomerFile(bytes.NewReader(workbook.Bytes()), "customers.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if preview.TotalRows != 2 || preview.ActiveRows != 1 || preview.InactiveRows != 1 {
		t.Fatalf("unexpected XLSX preview: %+v", preview)
	}
}

func TestPreviewCustomerFileRejectsUnsupportedExtension(t *testing.T) {
	_, err := PreviewCustomerFile(strings.NewReader("data"), "customers.xls")
	if err == nil || !strings.Contains(err.Error(), ".csv or .xlsx") {
		t.Fatalf("expected unsupported extension error, got %v", err)
	}
}

func TestParseImportDateSupportsCSVAndExcelFormats(t *testing.T) {
	for _, value := range []string{"2026-08-14", "8/14/2026", "08/14/2026", "8/14/26", "14-Aug-2026"} {
		if parsed := parseImportDate(value); parsed.Year() != 2026 || parsed.Month() != 8 || parsed.Day() != 14 {
			t.Fatalf("failed to parse %q: %v", value, parsed)
		}
	}
}

func TestImportCustomerCSVCreatesApprovedDistributionHierarchy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:customer_csv_distribution_import?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.NetworkRouter{},
		&models.POP{},
		&models.Agent{},
		&models.AgentPOP{},
		&models.Customer{},
		&models.User{},
		&models.Package{},
		&models.CustomerInternetAccount{},
		&models.Subscription{},
		&models.CustomerImportBatch{},
		&models.CustomerImportItem{},
	); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	router := models.NetworkRouter{Code: "RTR-IMPORT", Name: "Import Router", Host: "192.0.2.10", APIUsername: "test", Status: "ACTIVE"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	created, updated, err := SyncApprovedPackageCatalog()
	if err != nil || created != 40 || updated != 0 {
		t.Fatalf("unexpected initial package catalog sync: created=%d updated=%d err=%v", created, updated, err)
	}

	csvInput := strings.Join([]string{
		"ID,Username,Status,Package,POP,Name,Contact,Expire,B Cycle,Balance,J Date,C Date",
		"101,user-active,active,Pack:Little_P5,Kasundi &  Bagbaria,Active Customer,'01700000001',2026-09-14,14,120,2026-08-14,2026-08-14",
		"102,user-suspended,deactive,Pack:Little_P5,Head Office / Kalinagor,Suspended Customer,'01700000002',2026-09-15,15,80,2026-08-14,2026-08-14",
	}, "\n")
	batch, err := ImportCustomerCSV(strings.NewReader(csvInput), "customers.csv", router.ID)
	if err != nil {
		t.Fatal(err)
	}
	if batch.ImportedRows != 2 || batch.CreatedPackages != 0 || batch.CreatedPOPs != 15 || batch.CreatedAgents != 10 {
		t.Fatalf("unexpected import summary: %+v", batch)
	}

	var packageRow models.Package
	if err := db.Where("name = ?", "Little_P5").First(&packageRow).Error; err != nil {
		t.Fatal(err)
	}
	if packageRow.Price != 400 || packageRow.Commission != 200 || packageRow.MikroTikProfile != "Little_P5" {
		t.Fatalf("unexpected imported package: %+v", packageRow)
	}

	var customers []models.Customer
	if err := db.Order("customer_code").Find(&customers).Error; err != nil {
		t.Fatal(err)
	}
	if len(customers) != 2 || customers[0].PopID == nil || customers[0].AgentID == nil || customers[1].PopID == nil || customers[1].AgentID == nil {
		t.Fatalf("customers are missing POP/Agent mappings: %+v", customers)
	}

	var firstPOP models.POP
	var firstAgent models.Agent
	if err := db.First(&firstPOP, *customers[0].PopID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&firstAgent, *customers[0].AgentID).Error; err != nil {
		t.Fatal(err)
	}
	if firstPOP.Name != "Kasundi & Bagbaria" || firstAgent.Name != "Shukur Ali Biswas" || firstAgent.OpeningBalance != 13258.67 {
		t.Fatalf("unexpected POP/Agent mapping: pop=%+v agent=%+v", firstPOP, firstAgent)
	}

	var subscriptions []models.Subscription
	if err := db.Order("subscription_code").Find(&subscriptions).Error; err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 2 || subscriptions[0].Status != "ACTIVE" || subscriptions[1].Status != "SUSPENDED" {
		t.Fatalf("unexpected subscription statuses: %+v", subscriptions)
	}
	if subscriptions[0].InternetAccountID == nil || subscriptions[1].InternetAccountID == nil {
		t.Fatalf("subscriptions must link their canonical internet accounts: %+v", subscriptions)
	}
	var adoption models.CustomerInternetAccount
	if err := db.Where("pp_po_e_username = ?", "user-active").First(&adoption).Error; err != nil {
		t.Fatal(err)
	}
	if adoption.PPPoEPasswordEncrypted != "" || adoption.RouterID != router.ID || adoption.PackageID != packageRow.ID || adoption.MACAddress != "" || adoption.StaticIPAddress != "" || adoption.Status != "ACTIVE" {
		t.Fatalf("unexpected credential-less adoption account: %+v", adoption)
	}
}

func TestImportCustomerCSVEncryptsOptionalPasswordAndPreservesNetworkBindings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:customer_csv_password_import?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NetworkRouter{}, &models.POP{}, &models.Agent{}, &models.AgentPOP{}, &models.Customer{}, &models.User{}, &models.Package{}, &models.CustomerInternetAccount{}, &models.Subscription{}, &models.CustomerImportBatch{}, &models.CustomerImportItem{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	router := models.NetworkRouter{Code: "RTR-PASSWORD", Name: "Password Router", Host: "192.0.2.11", APIUsername: "test", Status: "ACTIVE"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := SyncApprovedPackageCatalog(); err != nil {
		t.Fatal(err)
	}
	const key = "0123456789abcdef0123456789abcdef"
	csvInput := "ID,Username,Status,Package,POP,Name,Contact,Expire,B Cycle,Password,IP Address,Mac,J Date\n201,password-user,active,Pack:Little_P5,Kasundi & Bagbaria,Password Customer,'01700000003',2026-09-20,20,source-password,198.51.100.42,AA:BB:CC:DD:EE:FF,2026-08-20"
	if _, err := ImportCustomerCSVWithCredentialKey(strings.NewReader(csvInput), "customers.csv", router.ID, key); err != nil {
		t.Fatal(err)
	}
	var account models.CustomerInternetAccount
	if err := db.Where("pp_po_e_username = ?", "password-user").First(&account).Error; err != nil {
		t.Fatal(err)
	}
	if account.PPPoEPasswordEncrypted == "" || account.PPPoEPasswordEncrypted == "source-password" {
		t.Fatalf("password was not encrypted")
	}
	password, err := security.DecryptSecret(account.PPPoEPasswordEncrypted, key)
	if err != nil || password != "source-password" {
		t.Fatalf("unexpected decrypted password %q: %v", password, err)
	}
	if account.StaticIPAddress != "198.51.100.42" || account.MACAddress != "AA:BB:CC:DD:EE:FF" || account.BillingDay != 20 || account.ExpiryDate == nil || account.ExpiryDate.Day() != 20 {
		t.Fatalf("network/lifecycle fields not preserved: %+v", account)
	}
	var subscription models.Subscription
	if err := db.Where("internet_account_id = ?", account.ID).First(&subscription).Error; err != nil {
		t.Fatal(err)
	}
	if subscription.PPPoEPasswordEncrypted != account.PPPoEPasswordEncrypted {
		t.Fatal("subscription compatibility credential was not linked to canonical account credential")
	}
}

func TestImportCustomerCSVRejectsPasswordWithoutCredentialKey(t *testing.T) {
	csvInput := "ID,Username,Status,Package,POP,Name,Contact,Expire,B Cycle,Password\n202,keyless-user,active,Pack:Little_P5,Kasundi & Bagbaria,Keyless,'01700000004',2026-09-20,20,source-password"
	if _, err := ImportCustomerCSV(strings.NewReader(csvInput), "customers.csv", 1); err == nil || !strings.Contains(err.Error(), "credential encryption key") {
		t.Fatalf("expected missing key rejection, got %v", err)
	}
}

func TestReadCustomerCSVCanonicalizesPasswordHeader(t *testing.T) {
	rows, err := readCustomerCSV(strings.NewReader("ID,Username,Status,Package,POP,Name,Contact,Expire,B Cycle,password\n1,case-user,active,Pack:Little_P5,Kasundi & Bagbaria,Case User,01700000000,2026-09-20,20,source-password"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["Password"] != "source-password" {
		t.Fatalf("password header was not canonicalized: %#v", rows)
	}
}

func TestImportCustomerCSVRejectsMissingSourceID(t *testing.T) {
	csvInput := "ID,Username,Status,Package,POP,Name,Contact,Expire,B Cycle\n,missing-id-user,active,Pack:Little_P5,Kasundi & Bagbaria,Missing ID,'01700000005',2026-09-20,20"
	if _, err := ImportCustomerCSV(strings.NewReader(csvInput), "customers.csv", 1); err == nil || !strings.Contains(err.Error(), "missing source ID") {
		t.Fatalf("expected missing source ID rejection, got %v", err)
	}
}

func TestParseImportDateAcceptsExcelDateTimeFormats(t *testing.T) {
	for _, value := range []string{"2026-09-02 00:00:00", "2/9/26 12:00:00 AM", "02/09/2026", "2/9/2026 00:00:00"} {
		parsed := parseImportDate(value)
		if parsed.IsZero() || parsed.Year() != 2026 || parsed.Month() != time.September || parsed.Day() != 2 {
			t.Fatalf("parseImportDate(%q) = %v", value, parsed)
		}
	}
}

func TestImportAddressMapsStructuredComponents(t *testing.T) {
	row := map[string]string{
		"Area":          "Magura Town",
		"Block":         "Block A",
		"Road Name":     "College Road",
		"Road No":       "12",
		"Building Name": "TS Tower",
		"Building No":   "7",
		"Flat":          "3B",
	}

	if got := importAddressParts(row, "Area", "Block", "Road Name", "Road No"); got != "Magura Town, Block A, College Road, 12" {
		t.Fatalf("unexpected road_or_area: %q", got)
	}

	if got := importAddressParts(row, "Building Name", "Building No", "Flat"); got != "TS Tower, 7, 3B" {
		t.Fatalf("unexpected village_or_holding: %q", got)
	}

	if got := importAddress(row); got != "Magura Town, Block A, College Road, 12, TS Tower, 7, 3B" {
		t.Fatalf("unexpected legacy address: %q", got)
	}
}
