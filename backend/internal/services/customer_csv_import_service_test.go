package services

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
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
		&models.Customer{},
		&models.Package{},
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
}
