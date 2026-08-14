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
	Mobile            string  `json:"mobile"`
	Address           string  `json:"address"`
	CommissionPercent float64 `json:"commission_percent"`
}

type UpdateAgentRequest struct {
	Name              string  `json:"name" binding:"required"`
	POPID             uint    `json:"pop_id" binding:"required"`
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
	ID                uint    `json:"id"`
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	POPID             uint    `json:"pop_id"`
	POPName           string  `json:"pop_name"`
	Mobile            string  `json:"mobile"`
	Address           string  `json:"address"`
	CommissionPercent float64 `json:"commission_percent"`
	Status            string  `json:"status"`
}

func ToAgentResponse(row models.Agent) AgentResponse {
	return AgentResponse{ID: row.ID, Code: row.Code, Name: row.Name, POPID: row.POPID, POPName: row.POP.Name, Mobile: row.Mobile, Address: row.Address, CommissionPercent: row.CommissionPercent, Status: row.Status}
}
