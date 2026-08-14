package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/gorm"
)

type AgentCollectionSummary struct {
	TotalAmount, TotalCommission float64
	VoidAmount                   float64
	Count, VoidCount             int64
}

func ListAgentCollections(agentID uint, status string) ([]models.AgentCollection, AgentCollectionSummary, error) {
	query := agentCollectionsQuery(agentID, status)
	var rows []models.AgentCollection
	if err := query.Preload("Agent").Preload("Customer").Preload("Payment").Order("collected_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, AgentCollectionSummary{}, err
	}
	var summary AgentCollectionSummary
	if err := agentCollectionsQuery(agentID, "").Select(`
		COALESCE(SUM(CASE WHEN status = 'ACTIVE' THEN 1 ELSE 0 END), 0) AS count,
		COALESCE(SUM(CASE WHEN status = 'ACTIVE' THEN amount ELSE 0 END), 0) AS total_amount,
		COALESCE(SUM(CASE WHEN status = 'ACTIVE' THEN commission_amount ELSE 0 END), 0) AS total_commission,
		COALESCE(SUM(CASE WHEN status = 'VOID' THEN 1 ELSE 0 END), 0) AS void_count,
		COALESCE(SUM(CASE WHEN status = 'VOID' THEN amount ELSE 0 END), 0) AS void_amount`).Scan(&summary).Error; err != nil {
		return nil, AgentCollectionSummary{}, err
	}
	return rows, summary, nil
}

func agentCollectionsQuery(agentID uint, status string) *gorm.DB {
	query := database.DB.Model(&models.AgentCollection{})
	if agentID > 0 {
		query = query.Where("agent_id = ?", agentID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	return query
}
