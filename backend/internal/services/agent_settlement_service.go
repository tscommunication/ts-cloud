package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type AgentSettlementBalance struct{ Earned, Paid, Payable float64 }

func GetAgentSettlementBalance(db *gorm.DB, agentID uint) (AgentSettlementBalance, error) {
	var balance AgentSettlementBalance
	if err := db.Model(&models.AgentCollection{}).Where("agent_id = ? AND status = ?", agentID, "ACTIVE").Select("COALESCE(SUM(commission_amount),0)").Scan(&balance.Earned).Error; err != nil {
		return balance, err
	}
	if err := db.Model(&models.AgentSettlement{}).Where("agent_id = ? AND status = ?", agentID, "PAID").Select("COALESCE(SUM(amount),0)").Scan(&balance.Paid).Error; err != nil {
		return balance, err
	}
	balance.Payable = balance.Earned - balance.Paid
	return balance, nil
}

func CreateAgentSettlement(row *models.AgentSettlement) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var agent models.Agent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&agent, row.AgentID).Error; err != nil {
			return errors.New("agent not found")
		}
		if row.Amount <= 0 {
			return errors.New("settlement amount must be greater than zero")
		}
		method := strings.ToUpper(strings.TrimSpace(row.Method))
		switch method {
		case "CASH":
		case "BKASH", "NAGAD", "ROCKET", "BANK":
			if strings.TrimSpace(row.TransactionID) == "" {
				return errors.New("transaction id required for " + method)
			}
		default:
			return errors.New("invalid settlement method")
		}
		balance, err := GetAgentSettlementBalance(tx, row.AgentID)
		if err != nil {
			return err
		}
		if row.Amount > balance.Payable {
			return fmt.Errorf("settlement amount exceeds payable commission %.2f", balance.Payable)
		}
		row.Method, row.Status = method, "PAID"
		if row.PaidAt.IsZero() {
			row.PaidAt = time.Now()
		}
		row.SettlementNo = fmt.Sprintf("TEMP-%d", time.Now().UnixNano())
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		row.SettlementNo = fmt.Sprintf("ASET-%06d", row.ID)
		return tx.Save(row).Error
	})
}

func VoidAgentSettlement(id uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var row models.AgentSettlement
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if row.Status != "PAID" {
			return errors.New("settlement is already void")
		}
		row.Status = "VOID"
		return tx.Save(&row).Error
	})
}

func ListAgentSettlements(agentID uint) ([]models.AgentSettlement, AgentSettlementBalance, error) {
	query := database.DB.Preload("Agent").Order("paid_at DESC, id DESC")
	if agentID > 0 {
		query = query.Where("agent_id = ?", agentID)
	}
	var rows []models.AgentSettlement
	if err := query.Find(&rows).Error; err != nil {
		return nil, AgentSettlementBalance{}, err
	}
	if agentID == 0 {
		return rows, AgentSettlementBalance{}, nil
	}
	balance, err := GetAgentSettlementBalance(database.DB, agentID)
	return rows, balance, err
}
