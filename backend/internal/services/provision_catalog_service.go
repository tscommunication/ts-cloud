package services

import (
	"fmt"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func ListProvisionCatalogPackages() ([]models.Package, error) {
	return repositories.ListActivePackages()
}

func ListProvisionCatalogRouters(
	role string,
	agentID uint,
) ([]models.NetworkRouter, error) {
	if role != "agent" {
		return repositories.ListActiveNetworkRouters()
	}

	if agentID == 0 {
		return nil, fmt.Errorf("agent account is not linked")
	}

	agent, err := repositories.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found")
	}

	if agent.Status != "ACTIVE" {
		return nil, fmt.Errorf("agent must be active")
	}

	popSet := map[uint]struct{}{}

	if agent.POPID > 0 {
		popSet[agent.POPID] = struct{}{}
	}

	for _, link := range agent.AgentPOPs {
		if link.POPID > 0 {
			popSet[link.POPID] = struct{}{}
		}
	}

	popIDs := make([]uint, 0, len(popSet))
	for id := range popSet {
		popIDs = append(popIDs, id)
	}

	return repositories.ListActiveNetworkRoutersByPOPIDs(popIDs)
}
