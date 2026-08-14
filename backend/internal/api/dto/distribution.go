package dto

import "github.com/tscommunication/ts-cloud/internal/models"

type CreatePOPRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	ManagerName string `json:"manager_name"`
	Mobile      string `json:"mobile"`
	Address     string `json:"address"`
}

type UpdatePOPRequest struct {
	Name        string `json:"name" binding:"required"`
	ManagerName string `json:"manager_name"`
	Mobile      string `json:"mobile"`
	Address     string `json:"address"`
}

type CreateAgentRequest struct {
	Code              string  `json:"code" binding:"required"`
	Name              string  `json:"name" binding:"required"`
	POPID             uint    `json:"pop_id" binding:"required"`
	POPIDs            []uint  `json:"pop_ids"`
	Mobile            string  `json:"mobile"`
	Address           string  `json:"address"`
	CommissionPercent float64 `json:"commission_percent"`
}

type UpdateAgentRequest struct {
	Name              string  `json:"name" binding:"required"`
	POPID             uint    `json:"pop_id" binding:"required"`
	POPIDs            []uint  `json:"pop_ids"`
	Mobile            string  `json:"mobile"`
	Address           string  `json:"address"`
	CommissionPercent float64 `json:"commission_percent"`
}

type UpdateDistributionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=ACTIVE INACTIVE"`
}

type POPResponse struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	ManagerName string `json:"manager_name"`
	Mobile      string `json:"mobile"`
	Address     string `json:"address"`
	Status      string `json:"status"`
}

func ToPOPResponse(row models.POP) POPResponse {
	return POPResponse{ID: row.ID, Code: row.Code, Name: row.Name, ManagerName: row.ManagerName, Mobile: row.Mobile, Address: row.Address, Status: row.Status}
}

type AgentResponse struct {
	ID                uint     `json:"id"`
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	POPID             uint     `json:"pop_id"`
	POPName           string   `json:"pop_name"`
	POPIDs            []uint   `json:"pop_ids"`
	POPNames          []string `json:"pop_names"`
	Mobile            string   `json:"mobile"`
	Address           string   `json:"address"`
	CommissionPercent float64  `json:"commission_percent"`
	OpeningBalance    float64  `json:"opening_balance"`
	SourceReference   string   `json:"source_reference"`
	Status            string   `json:"status"`
}

func ToAgentResponse(row models.Agent) AgentResponse {
	popIDs, popNames := make([]uint, 0, len(row.AgentPOPs)), make([]string, 0, len(row.AgentPOPs))
	seen := map[uint]bool{}
	for _, link := range row.AgentPOPs {
		if seen[link.POPID] {
			continue
		}
		seen[link.POPID] = true
		popIDs = append(popIDs, link.POPID)
		popNames = append(popNames, link.POP.Name)
	}
	if !seen[row.POPID] {
		popIDs = append(popIDs, row.POPID)
		popNames = append(popNames, row.POP.Name)
	}
	return AgentResponse{ID: row.ID, Code: row.Code, Name: row.Name, POPID: row.POPID, POPName: row.POP.Name, POPIDs: popIDs, POPNames: popNames, Mobile: row.Mobile, Address: row.Address, CommissionPercent: row.CommissionPercent, OpeningBalance: row.OpeningBalance, SourceReference: row.SourceReference, Status: row.Status}
}
