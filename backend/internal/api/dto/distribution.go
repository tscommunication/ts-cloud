package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

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
	PackageIDs        []uint  `json:"package_ids"`
	RouterIDs         []uint  `json:"router_ids"`
	NetworkDeviceIDs  []uint  `json:"network_device_ids"`
	Mobile            string  `json:"mobile"`
	Address           string  `json:"address"`
	CommissionPercent float64 `json:"commission_percent"`
}

type UpdateAgentRequest struct {
	Name              string  `json:"name" binding:"required"`
	POPID             uint    `json:"pop_id" binding:"required"`
	POPIDs            []uint  `json:"pop_ids"`
	PackageIDs        []uint  `json:"package_ids"`
	RouterIDs         []uint  `json:"router_ids"`
	NetworkDeviceIDs  []uint  `json:"network_device_ids"`
	Mobile            string  `json:"mobile"`
	Address           string  `json:"address"`
	CommissionPercent float64 `json:"commission_percent"`
}

type UpdateAgentPackagesRequest struct {
	PackageIDs []uint `json:"package_ids"`
}

type UpdateAgentPermissionsRequest struct {
	PackageIDs       []uint `json:"package_ids"`
	RouterIDs        []uint `json:"router_ids"`
	NetworkDeviceIDs []uint `json:"network_device_ids"`
}

type UpdateDistributionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=ACTIVE INACTIVE"`
}

type POPResponse struct {
	ID              uint       `json:"id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	ManagerName     string     `json:"manager_name"`
	Mobile          string     `json:"mobile"`
	Address         string     `json:"address"`
	SourceReference string     `json:"source_reference"`
	Status          string     `json:"status"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

func ToPOPResponse(row models.POP) POPResponse {
	var deletedAt *time.Time

	if row.DeletedAt.Valid {
		value := row.DeletedAt.Time
		deletedAt = &value
	}

	return POPResponse{
		ID:              row.ID,
		Code:            row.Code,
		Name:            row.Name,
		ManagerName:     row.ManagerName,
		Mobile:          row.Mobile,
		Address:         row.Address,
		SourceReference: row.SourceReference,
		Status:          row.Status,
		DeletedAt:       deletedAt,
	}
}

type AgentResponse struct {
	ID                 uint       `json:"id"`
	Code               string     `json:"code"`
	Name               string     `json:"name"`
	POPID              uint       `json:"pop_id"`
	POPName            string     `json:"pop_name"`
	POPIDs             []uint     `json:"pop_ids"`
	POPNames           []string   `json:"pop_names"`
	PackageIDs         []uint     `json:"package_ids"`
	PackageNames       []string   `json:"package_names"`
	RouterIDs          []uint     `json:"router_ids"`
	RouterNames        []string   `json:"router_names"`
	NetworkDeviceIDs   []uint     `json:"network_device_ids"`
	NetworkDeviceNames []string   `json:"network_device_names"`
	Mobile             string     `json:"mobile"`
	Address            string     `json:"address"`
	CommissionPercent  float64    `json:"commission_percent"`
	OpeningBalance     float64    `json:"opening_balance"`
	SourceReference    string     `json:"source_reference"`
	Status             string     `json:"status"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
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
	packageIDs := make([]uint, 0, len(row.AgentPackages))
	packageNames := make([]string, 0, len(row.AgentPackages))
	for _, link := range row.AgentPackages {
		packageIDs = append(packageIDs, link.PackageID)
		packageNames = append(packageNames, link.Package.PackageCode+" — "+link.Package.Name)
	}
	routerIDs, routerNames := make([]uint, 0, len(row.AgentRouters)), make([]string, 0, len(row.AgentRouters))
	for _, link := range row.AgentRouters {
		routerIDs = append(routerIDs, link.RouterID)
		routerNames = append(routerNames, link.Router.Code+" — "+link.Router.Name)
	}

	networkDeviceIDs := make([]uint, 0, len(row.AgentNetworkDevices))
	networkDeviceNames := make([]string, 0, len(row.AgentNetworkDevices))
	for _, link := range row.AgentNetworkDevices {
		networkDeviceIDs = append(networkDeviceIDs, link.NetworkDeviceID)
		networkDeviceNames = append(
			networkDeviceNames,
			link.NetworkDevice.Code+" — "+link.NetworkDevice.Name,
		)
	}
	var deletedAt *time.Time

	if row.DeletedAt.Valid {
		value := row.DeletedAt.Time
		deletedAt = &value
	}

	return AgentResponse{
		ID:                 row.ID,
		Code:               row.Code,
		Name:               row.Name,
		POPID:              row.POPID,
		POPName:            row.POP.Name,
		POPIDs:             popIDs,
		POPNames:           popNames,
		PackageIDs:         packageIDs,
		PackageNames:       packageNames,
		RouterIDs:          routerIDs,
		RouterNames:        routerNames,
		NetworkDeviceIDs:   networkDeviceIDs,
		NetworkDeviceNames: networkDeviceNames,
		Mobile:             row.Mobile,
		Address:            row.Address,
		CommissionPercent:  row.CommissionPercent,
		OpeningBalance:     row.OpeningBalance,
		SourceReference:    row.SourceReference,
		Status:             row.Status,
		DeletedAt:          deletedAt,
	}
}
