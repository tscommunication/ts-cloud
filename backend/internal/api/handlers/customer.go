package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetCustomerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	if !canAccessCustomer(c, customer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	c.JSON(http.StatusOK, dto.ToCustomerResponse(*customer))
}

func GetCustomerSummary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	if !canAccessCustomer(c, customer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	summary, err := services.GetCustomerSummary(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customer summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions":        summary.Subscriptions,
		"active_subscriptions": summary.ActiveSubscriptions,
		"invoices":             summary.Invoices,
		"outstanding_amount":   summary.OutstandingAmount,
		"successful_payments":  summary.SuccessfulPayments,
		"total_paid":           summary.TotalPaid,
		"cancelled_invoices":   summary.CancelledInvoices,
		"voided_payments":      summary.VoidedPayments,
		"voided_amount":        summary.VoidedAmount,
	})
}

func GetCustomers(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page must be a positive integer"})
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page size must be between 1 and 100"})
		return
	}

	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if status != "" && status != "ACTIVE" && status != "INACTIVE" && status != "ARCHIVED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be ACTIVE, INACTIVE or ARCHIVED"})
		return
	}

	agentID := uint(0)
	if c.GetString("role") == "agent" {
		agentID = c.GetUint("agent_id")
		if agentID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Agent account is not linked"})
			return
		}
	}
	customers, total, err := services.ListCustomers(repositories.CustomerListParams{
		Search:   c.Query("search"),
		Status:   status,
		Page:     page,
		PageSize: pageSize,
		AgentID:  agentID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get customers",
		})
		return
	}

	response := make([]dto.CustomerResponse, 0, len(customers))

	for _, customer := range customers {
		response = append(response, dto.ToCustomerResponse(customer))
	}

	c.JSON(http.StatusOK, gin.H{
		"count":     total,
		"customers": response,
		"page":      page,
		"page_size": pageSize,
	})
}

func canAccessCustomer(c *gin.Context, customer *models.Customer) bool {
	if c.GetString("role") != "agent" {
		return true
	}
	agentID := c.GetUint("agent_id")
	return agentID > 0 && customer.AgentID != nil && *customer.AgentID == agentID
}

func CreateCustomer(c *gin.Context) {
	var req dto.CreateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}
	if err := services.ValidateCustomerDistribution(req.PopID, req.AgentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate Customer Code
	customerCode := "CUS-000001"

	lastCustomer, err := services.GetLastCustomer()
	if err == nil {
		customerCode = fmt.Sprintf("CUS-%06d", lastCustomer.ID+1)
	}

	customer := models.Customer{
		CustomerCode:     customerCode,
		FullName:         req.FullName,
		Mobile:           req.Mobile,
		FatherName:       req.FatherName,
		MotherName:       req.MotherName,
		AltMobile:        req.AltMobile,
		Email:            req.Email,
		NID:              strings.TrimSpace(req.NID),
		Country:          strings.TrimSpace(req.Country),
		Division:         strings.TrimSpace(req.Division),
		District:         strings.TrimSpace(req.District),
		Upazila:          strings.TrimSpace(req.Upazila),
		PostOffice:       strings.TrimSpace(req.PostOffice),
		RoadOrArea:       strings.TrimSpace(req.RoadOrArea),
		VillageOrHolding: strings.TrimSpace(req.VillageOrHolding),
		Union:            strings.TrimSpace(req.Union),
		Village:          strings.TrimSpace(req.Village),
		Address:          strings.TrimSpace(req.Address),
		BillingDay:       req.BillingDay,
		PopID:            req.PopID,
		AgentID:          req.AgentID,
		Status:           "ACTIVE",
	}

	if customer.BillingDay == 0 {
		customer.BillingDay = 1
	}

	if err := services.CreateCustomer(&customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.ToCustomerResponse(customer))
}

func UpdateCustomer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var req dto.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer data"})
		return
	}
	if err := services.ValidateCustomerDistribution(req.PopID, req.AgentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	customer.FullName = strings.TrimSpace(req.FullName)
	customer.Mobile = strings.TrimSpace(req.Mobile)
	customer.FatherName = strings.TrimSpace(req.FatherName)
	customer.MotherName = strings.TrimSpace(req.MotherName)
	customer.AltMobile = strings.TrimSpace(req.AltMobile)
	customer.Email = strings.TrimSpace(req.Email)
	customer.NID = strings.TrimSpace(req.NID)
	customer.Country = strings.TrimSpace(req.Country)
	customer.Division = strings.TrimSpace(req.Division)
	customer.District = strings.TrimSpace(req.District)
	customer.Upazila = strings.TrimSpace(req.Upazila)
	customer.PostOffice = strings.TrimSpace(req.PostOffice)
	customer.RoadOrArea = strings.TrimSpace(req.RoadOrArea)
	customer.VillageOrHolding = strings.TrimSpace(req.VillageOrHolding)
	customer.Union = strings.TrimSpace(req.Union)
	customer.Village = strings.TrimSpace(req.Village)
	customer.Address = strings.TrimSpace(req.Address)
	customer.BillingDay = req.BillingDay
	customer.PopID = req.PopID
	customer.AgentID = req.AgentID

	if err := services.UpdateCustomer(customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer"})
		return
	}

	c.JSON(http.StatusOK, dto.ToCustomerResponse(*customer))
}

func UpdateCustomerStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var req dto.UpdateCustomerStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be ACTIVE or INACTIVE"})
		return
	}

	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	if customer.Status == "ARCHIVED" {
		c.JSON(http.StatusConflict, gin.H{"error": "Archived customers cannot be activated or deactivated"})
		return
	}

	customer.Status = req.Status
	if err := services.UpdateCustomer(customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer status"})
		return
	}

	c.JSON(http.StatusOK, dto.ToCustomerResponse(*customer))
}

func ArchiveCustomer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	if customer.Status == "ARCHIVED" {
		c.JSON(http.StatusConflict, gin.H{"error": "Customer is already archived"})
		return
	}

	if err := services.ArchiveCustomer(customer); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ToCustomerResponse(*customer))
}
