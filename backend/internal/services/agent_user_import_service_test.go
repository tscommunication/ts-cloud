package services

import (
	"bytes"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func agentUserWorkbook(t *testing.T) []byte {
	t.Helper()
	book := excelize.NewFile()
	defer book.Close()
	sheet := book.GetSheetName(0)
	rows := [][]interface{}{
		{"ID", "Agents name", "user name", "Password", "Company", "Role", "Active State", "Accounting"},
		{"", "Md. Tariqul Islam", "owner@tscl", "secure-pass-1", "TS", "superadmin", "Active", "yes"},
		{"", "Md. Sohrab Hossain", "santo@tscl", "secure-pass-2", "", "Agent", "Active", "yes"},
		{"", "Unmatched Person", "skip@tscl", "secure-pass-3", "", "Agent", "Active", "yes"},
	}
	for index, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		if err != nil {
			t.Fatal(err)
		}
		if err := book.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	if err := book.Write(&output); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestAgentUserWorkbookPreviewAndImport(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent_user_import?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.POP{}, &models.Agent{}, &models.User{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	pop := models.POP{Code: "POP-1", Name: "Test POP", Status: "ACTIVE"}
	if err := db.Create(&pop).Error; err != nil {
		t.Fatal(err)
	}
	agent := models.Agent{Code: "AG-1", Name: "Md. Sohrab Hossain", POPID: pop.ID, Status: "ACTIVE"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}

	workbook := agentUserWorkbook(t)
	preview, err := PreviewAgentUserWorkbook(bytes.NewReader(workbook), "Agents.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if preview.ReadyRows != 2 || preview.CreateRows != 2 || preview.SkippedRows != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}

	result, err := ImportAgentUserWorkbook(bytes.NewReader(workbook), "Agents.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if result.CreatedRows != 2 || result.SkippedRows != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	var imported models.User
	if err := db.Where("username = ?", "santo@tscl").First(&imported).Error; err != nil {
		t.Fatal(err)
	}
	if imported.AgentID == nil || *imported.AgentID != agent.ID || imported.Role != "agent" {
		t.Fatalf("agent link was not imported: %+v", imported)
	}
	if bcrypt.CompareHashAndPassword([]byte(imported.Password), []byte("secure-pass-2")) != nil {
		t.Fatal("imported password is not a valid bcrypt hash")
	}
	var skipped int64
	if err := db.Model(&models.User{}).Where("username = ?", "skip@tscl").Count(&skipped).Error; err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatal("unmatched Agent account should be skipped")
	}
}
