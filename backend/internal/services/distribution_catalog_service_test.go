package services

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSyncApprovedDistributionCatalogIsCompleteAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:approved_distribution_catalog?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.POP{}, &models.Agent{}, &models.AgentPOP{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	first, err := SyncApprovedDistributionCatalog()
	if err != nil || first.CreatedPOPs != 15 || first.CreatedAgents != 10 {
		t.Fatalf("unexpected first sync: result=%+v err=%v", first, err)
	}
	second, err := SyncApprovedDistributionCatalog()
	if err != nil || second.CreatedPOPs != 0 || second.CreatedAgents != 0 || second.UpdatedPOPs != 15 || second.UpdatedAgents != 10 {
		t.Fatalf("unexpected second sync: result=%+v err=%v", second, err)
	}

	var popCount, agentCount, deleteUserCount int64
	if err := db.Model(&models.POP{}).Count(&popCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Agent{}).Count(&agentCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.POP{}).Where("LOWER(name) = ?", "delete user").Count(&deleteUserCount).Error; err != nil {
		t.Fatal(err)
	}
	if popCount != 15 || agentCount != 10 || deleteUserCount != 0 {
		t.Fatalf("unexpected catalog counts: pops=%d agents=%d delete_user=%d", popCount, agentCount, deleteUserCount)
	}

	var headOffice models.POP
	if err := db.Where("name = ?", "Head Office / Kalinagor").First(&headOffice).Error; err != nil {
		t.Fatal(err)
	}
	if headOffice.ManagerName != "Md. Tariqul Islam" || headOffice.Code != "CAT-POP-H1" {
		t.Fatalf("unexpected Head Office mapping: %+v", headOffice)
	}
	var agent models.Agent
	if err := db.Where("name = ?", "Shukur Ali Biswas").First(&agent).Error; err != nil {
		t.Fatal(err)
	}
	if agent.OpeningBalance != 13258.67 || agent.Mobile != "01710040852" || agent.SourceReference != "MANAGER-11; TYPE=own" {
		t.Fatalf("unexpected synchronized agent: %+v", agent)
	}
	var rony models.Agent
	if err := db.Preload("AgentPOPs.POP").Where("name = ?", "Raichul Islam (Rony)").First(&rony).Error; err != nil {
		t.Fatal(err)
	}
	if len(rony.AgentPOPs) != 2 {
		t.Fatalf("expected Rony to have 2 POP locations, got %+v", rony.AgentPOPs)
	}
	linkedNames := map[string]bool{}
	for _, link := range rony.AgentPOPs {
		linkedNames[link.POP.Name] = true
	}
	if !linkedNames["Nakol OLT"] || !linkedNames["Nakol MC"] {
		t.Fatalf("unexpected Rony POP locations: %+v", linkedNames)
	}
	var nakolMC models.POP
	if err := db.Where("name = ?", "Nakol MC").First(&nakolMC).Error; err != nil {
		t.Fatal(err)
	}
	if err := ValidateCustomerDistribution(&nakolMC.ID, &rony.ID); err != nil {
		t.Fatalf("expected secondary POP assignment to validate: %v", err)
	}
}
