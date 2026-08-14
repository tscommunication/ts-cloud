package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/gorm"
)

type AgentCollectionSummary struct {
	TotalAmount, TotalCommission float64
	Count                        int64
}

func ListAgentCollections(agentID uint, status string) ([]models.AgentCollection, AgentCollectionSummary, error) {
	query := agentCollectionsQuery(agentID, status)
	var rows []models.AgentCollection
	if err := query.Preload("Agent").Preload("Customer").Preload("Payment").Order("collected_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, AgentCollectionSummary{}, err
	}
	var summary AgentCollectionSummary
	if err := agentCollectionsQuery(agentID, status).Select("COUNT(*) AS count, COALESCE(SUM(amount),0) AS total_amount, COALESCE(SUM(commission_amount),0) AS total_commission").Scan(&summary).Error; err != nil {
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
