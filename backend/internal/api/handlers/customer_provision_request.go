package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func provisionRequestID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provision request ID"})
		return 0, false
	}

	return uint(id), true
}

func canAccessProvisionRequest(
	c *gin.Context,
	request *models.CustomerProvisionRequest,
) bool {
	if c.GetString("role") != "agent" {
		return true
	}

	agentID := c.GetUint("agent_id")
	return agentID > 0 &&
		request.AgentID != nil &&
		*request.AgentID == agentID
}

func CreateCustomerProvisionRequest(
	cfg *config.Config,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateCustomerProvisionRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid provision request",
			})
			return
		}

		var activationDate time.Time
		if err := services.ValidateAgentPackage(c.GetUint("agent_id"), req.PackageID); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		if value := strings.TrimSpace(req.ActivationDate); value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Activation date must use YYYY-MM-DD format",
				})
				return
			}
			activationDate = parsed
		}

		request := models.CustomerProvisionRequest{
			FullName:         req.FullName,
			Mobile:           req.Mobile,
			FatherName:       req.FatherName,
			MotherName:       req.MotherName,
			AltMobile:        req.AltMobile,
			Email:            req.Email,
			NID:              req.NID,
			Country:          req.Country,
			Division:         req.Division,
			District:         req.District,
			Upazila:          req.Upazila,
			PostOffice:       req.PostOffice,
			PostalCode:       req.PostalCode,
			RoadOrArea:       req.RoadOrArea,
			VillageOrHolding: req.VillageOrHolding,
			PackageID:        req.PackageID,
			RouterID:         req.RouterID,
			PPPoEUsername:    req.PPPoEUsername,
			BillingDay:       req.BillingDay,
			ActivationDate:   activationDate,
			Remarks:          req.Remarks,
		}

		if err := services.SetProvisionRequestPPPoEPassword(
			&request,
			req.PPPoEPassword,
			cfg.CredentialKey,
		); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		if err := services.CreateAgentCustomerProvisionRequest(
			&request,
			c.GetUint("agent_id"),
			c.GetUint("user_id"),
			time.Now(),
		); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(
			http.StatusCreated,
			dto.ToCustomerProvisionRequestResponse(request),
		)
	}
}

func GetCustomerProvisionRequests(c *gin.Context) {
	agentID := uint(0)

	if c.GetString("role") == "agent" {
		agentID = c.GetUint("agent_id")

		if agentID == 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Agent account is not linked",
			})
			return
		}
	}

	rows, err := services.ListCustomerProvisionRequests(
		c.Query("status"),
		agentID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := make(
		[]dto.CustomerProvisionRequestResponse,
		0,
		len(rows),
	)

	for _, row := range rows {
		response = append(
			response,
			dto.ToCustomerProvisionRequestResponse(row),
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(response),
		"requests": response,
	})
}

func GetCustomerProvisionRequest(c *gin.Context) {
	id, ok := provisionRequestID(c)
	if !ok {
		return
	}

	request, err := services.GetCustomerProvisionRequestByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Provision request not found",
		})
		return
	}

	if !canAccessProvisionRequest(c, request) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Permission denied",
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerProvisionRequestResponse(*request),
	)
}

func ApproveCustomerProvisionRequest(
	cfg *config.Config,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := provisionRequestID(c)
		if !ok {
			return
		}

		request, err := services.GetCustomerProvisionRequestByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Provision request not found",
			})
			return
		}

		if err := services.ApproveCustomerProvisionRequest(
			request,
			c.GetUint("user_id"),
			time.Now(),
			cfg.CredentialKey,
		); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(
			http.StatusOK,
			dto.ToCustomerProvisionRequestResponse(*request),
		)
	}
}

func RejectCustomerProvisionRequest(c *gin.Context) {
	id, ok := provisionRequestID(c)
	if !ok {
		return
	}

	var req dto.RejectCustomerProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Rejection reason is required",
		})
		return
	}

	request, err := services.GetCustomerProvisionRequestByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Provision request not found",
		})
		return
	}

	if err := services.RejectCustomerProvisionRequest(
		request,
		c.GetUint("user_id"),
		req.Reason,
		time.Now(),
	); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerProvisionRequestResponse(*request),
	)
}
