package services

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateAgentMovesOwnershipAndPreservesHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent_migration?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.POP{}, &models.Agent{}, &models.AgentPOP{}, &models.Customer{}, &models.User{}, &models.AgentCollection{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	pop1, pop2 := models.POP{Code: "P1", Name: "POP 1", Status: "ACTIVE"}, models.POP{Code: "P2", Name: "POP 2", Status: "ACTIVE"}
	if err := db.Create(&pop1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pop2).Error; err != nil {
		t.Fatal(err)
	}
	source, target := models.Agent{Code: "A1", Name: "Source", POPID: pop1.ID, Status: "ACTIVE"}, models.Agent{Code: "A2", Name: "Target", POPID: pop2.ID, Status: "ACTIVE"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AgentPOP{AgentID: source.ID, POPID: pop1.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AgentPOP{AgentID: target.ID, POPID: pop2.ID}).Error; err != nil {
		t.Fatal(err)
	}
	customer := models.Customer{CustomerCode: "C1", FullName: "Customer", Mobile: "1", AgentID: &source.ID, PopID: &pop1.ID, Status: "ACTIVE", BillingDay: 1}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Name: "Agent User", Username: "agent", Email: "agent@example.com", Password: "hash", Role: "agent", Active: true, AgentID: &source.ID}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	history := models.AgentCollection{AgentID: source.ID, CustomerID: customer.ID, PaymentID: 1, Amount: 100, CommissionRate: 10, CommissionAmount: 10, Status: "ACTIVE", CollectedAt: time.Now()}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}

	result, err := MigrateAgent(source.ID, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Customers != 1 || result.Users != 1 || result.POPs != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if err := db.First(&customer, customer.ID).Error; err != nil {
		t.Fatal(err)
	}
	if customer.AgentID == nil || *customer.AgentID != target.ID {
		t.Fatal("customer was not migrated")
	}
	if err := db.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.AgentID == nil || *user.AgentID != target.ID {
		t.Fatal("login user was not migrated")
	}
	if err := db.First(&history, history.ID).Error; err != nil {
		t.Fatal(err)
	}
	if history.AgentID != source.ID {
		t.Fatal("financial history must remain on source agent")
	}
	var sourceCount int64
	if err := db.Model(&models.Agent{}).Where("id = ?", source.ID).Count(&sourceCount).Error; err != nil {
		t.Fatal(err)
	}
	if sourceCount != 0 {
		t.Fatal("source agent should be soft deleted")
	}
	var targetPOPCount int64
	if err := db.Model(&models.AgentPOP{}).Where("agent_id = ?", target.ID).Count(&targetPOPCount).Error; err != nil {
		t.Fatal(err)
	}
	if targetPOPCount != 2 {
		t.Fatalf("expected two target POP memberships, got %d", targetPOPCount)
	}
}
