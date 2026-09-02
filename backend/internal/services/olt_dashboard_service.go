package services

import "github.com/tscommunication/ts-cloud/internal/repositories"

func GetOLTDashboard(
	agentID uint,
) (*repositories.OLTDashboard, error) {
	return repositories.GetOLTDashboard(agentID)
}
