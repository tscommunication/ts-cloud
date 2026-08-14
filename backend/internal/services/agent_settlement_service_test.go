package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestAgentSettlementBalanceAndVoid(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Agent{}, &models.AgentCollection{}, &models.AgentSettlement{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
	agent := models.Agent{Code: "AG-001", Name: "Agent", POPID: 1, Status: "ACTIVE"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}
	collection := models.AgentCollection{AgentID: agent.ID, CustomerID: 1, PaymentID: 1, Amount: 500, CommissionRate: 30, CommissionAmount: 150, Status: "ACTIVE", CollectedAt: time.Now()}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatal(err)
	}
	settlement := models.AgentSettlement{AgentID: agent.ID, Amount: 100, Method: "CASH"}
	if err := CreateAgentSettlement(&settlement); err != nil {
		t.Fatal(err)
	}
	balance, err := GetAgentSettlementBalance(db, agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if balance.Earned != 150 || balance.Paid != 100 || balance.Payable != 50 {
		t.Fatalf("unexpected balance: %+v", balance)
	}
	if err := CreateAgentSettlement(&models.AgentSettlement{AgentID: agent.ID, Amount: 60, Method: "CASH"}); err == nil {
		t.Fatal("expected over-settlement error")
	}
	if err := VoidAgentSettlement(settlement.ID); err != nil {
		t.Fatal(err)
	}
	balance, err = GetAgentSettlementBalance(db, agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if balance.Paid != 0 || balance.Payable != 150 {
		t.Fatalf("unexpected balance after void: %+v", balance)
	}
}
