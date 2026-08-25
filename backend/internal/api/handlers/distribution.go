package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func distributionID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return 0, false
	}
	return uint(id), true
}

func GetPOPs(c *gin.Context) {
	rows, err := services.ListPOPs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get POPs"})
		return
	}
	response := make([]dto.POPResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, dto.ToPOPResponse(row))
	}
	c.JSON(http.StatusOK, gin.H{"count": len(response), "pops": response})
}

func GetArchivedPOPs(c *gin.Context) {
	rows, err := services.ListArchivedPOPs()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to get archived POPs"},
		)
		return
	}

	response := make([]dto.POPResponse, 0, len(rows))

	for _, row := range rows {
		response = append(
			response,
			dto.ToPOPResponse(row),
		)
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"count": len(response),
			"pops":  response,
		},
	)
}

func GetPOP(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	row, err := services.GetPOP(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "POP not found"})
		return
	}
	c.JSON(http.StatusOK, dto.ToPOPResponse(*row))
}

func CreatePOP(c *gin.Context) {
	var req dto.CreatePOPRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid POP data"})
		return
	}
	row := models.POP{Code: req.Code, Name: req.Name, ManagerName: strings.TrimSpace(req.ManagerName), Mobile: strings.TrimSpace(req.Mobile), Address: strings.TrimSpace(req.Address)}
	if err := services.CreatePOP(&row); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dto.ToPOPResponse(row))
}

func UpdatePOP(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	var req dto.UpdatePOPRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid POP data"})
		return
	}
	row, err := services.GetPOP(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "POP not found"})
		return
	}
	row.Name, row.ManagerName, row.Mobile, row.Address = req.Name, strings.TrimSpace(req.ManagerName), strings.TrimSpace(req.Mobile), strings.TrimSpace(req.Address)
	if err := services.UpdatePOP(row); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.ToPOPResponse(*row))
}

func UpdatePOPStatus(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	var req dto.UpdateDistributionStatusRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be ACTIVE or INACTIVE"})
		return
	}
	row, err := services.GetPOP(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "POP not found"})
		return
	}
	row.Status = req.Status
	if err := services.UpdatePOP(row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update POP"})
		return
	}
	c.JSON(http.StatusOK, dto.ToPOPResponse(*row))
}

func GetAgents(c *gin.Context) {
	var popID uint
	if value := c.Query("pop_id"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid POP ID"})
			return
		}
		popID = uint(parsed)
	}
	rows, err := services.ListAgents(popID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agents"})
		return
	}
	response := make([]dto.AgentResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, dto.ToAgentResponse(row))
	}
	c.JSON(http.StatusOK, gin.H{"count": len(response), "agents": response})
}

func GetArchivedAgents(c *gin.Context) {
	rows, err := services.ListArchivedAgents()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to get archived agents"},
		)
		return
	}

	response := make([]dto.AgentResponse, 0, len(rows))

	for _, row := range rows {
		response = append(
			response,
			dto.ToAgentResponse(row),
		)
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"count":  len(response),
			"agents": response,
		},
	)
}

func GetAgent(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	row, err := services.GetAgent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}
	c.JSON(http.StatusOK, dto.ToAgentResponse(*row))
}

func CreateAgent(c *gin.Context) {
	var req dto.CreateAgentRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent data"})
		return
	}
	row := models.Agent{Code: req.Code, Name: req.Name, POPID: req.POPID, Mobile: strings.TrimSpace(req.Mobile), Address: strings.TrimSpace(req.Address), CommissionPercent: req.CommissionPercent}
	if err := services.CreateAgentWithPOPs(&row, req.POPIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.SetAgentRouters(row.ID, req.RouterIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.SetAgentNetworkDevices(
		row.ID,
		req.NetworkDeviceIDs,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := services.GetAgent(row.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent created but could not be reloaded"})
		return
	}
	c.JSON(http.StatusCreated, dto.ToAgentResponse(*created))
}

func UpdateAgent(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	var req dto.UpdateAgentRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent data"})
		return
	}
	row, err := services.GetAgent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}
	row.Name, row.POPID, row.Mobile, row.Address, row.CommissionPercent = req.Name, req.POPID, strings.TrimSpace(req.Mobile), strings.TrimSpace(req.Address), req.CommissionPercent
	if err := services.UpdateAgentWithPOPs(row, req.POPIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.SetAgentRouters(row.ID, req.RouterIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.SetAgentNetworkDevices(
		row.ID,
		req.NetworkDeviceIDs,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := services.GetAgent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent updated but could not be reloaded"})
		return
	}
	c.JSON(http.StatusOK, dto.ToAgentResponse(*updated))
}

func UpdateAgentPackages(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	var req dto.UpdateAgentPackagesRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package_ids is required"})
		return
	}
	if _, err := services.GetAgent(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}
	if err := services.SetAgentPackages(id, req.PackageIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := services.GetAgent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Packages saved but agent could not be reloaded"})
		return
	}
	c.JSON(http.StatusOK, dto.ToAgentResponse(*updated))
}

func UpdateAgentPermissions(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	var req dto.UpdateAgentPermissionsRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package_ids, router_ids and network_device_ids are required"})
		return
	}
	if _, err := services.GetAgent(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}
	if err := services.SetAgentPackages(id, req.PackageIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.SetAgentRouters(id, req.RouterIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.SetAgentNetworkDevices(
		id,
		req.NetworkDeviceIDs,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := services.GetAgent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Permissions saved but agent could not be reloaded"})
		return
	}
	c.JSON(http.StatusOK, dto.ToAgentResponse(*updated))
}

func UpdateAgentStatus(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	var req dto.UpdateDistributionStatusRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be ACTIVE or INACTIVE"})
		return
	}
	row, err := services.GetAgent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}
	row.Status = req.Status
	if err := services.UpdateAgent(row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent"})
		return
	}
	c.JSON(http.StatusOK, dto.ToAgentResponse(*row))
}

func MigrateAgent(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	var req struct {
		TargetAgentID uint `json:"target_agent_id" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target agent is required"})
		return
	}
	result, err := services.MigrateAgent(id, req.TargetAgentID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func DeleteAgent(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}
	if err := services.DeleteAgent(id); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func RestoreAgent(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}

	row, err := services.RestoreAgent(id)
	if err != nil {
		c.JSON(
			http.StatusConflict,
			gin.H{"error": err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToAgentResponse(*row),
	)
}

func MigratePOP(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}

	var req struct {
		TargetPOPID uint `json:"target_pop_id" binding:"required"`
	}

	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Target POP is required",
		})
		return
	}

	result, err := services.MigratePOP(id, req.TargetPOPID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func DeletePOP(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}

	if err := services.DeletePOP(id); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

func RestorePOP(c *gin.Context) {
	id, ok := distributionID(c)
	if !ok {
		return
	}

	row, err := services.RestorePOP(id)
	if err != nil {
		c.JSON(
			http.StatusConflict,
			gin.H{"error": err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToPOPResponse(*row),
	)
}
