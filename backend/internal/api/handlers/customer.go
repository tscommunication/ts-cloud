package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/config"
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
	view := strings.ToUpper(strings.TrimSpace(c.Query("view")))
	validViews := map[string]bool{"": true, "EXPIRED": true, "PENDING": true, "RECENT": true, "DISABLED": true, "ONLINE": true, "OFFLINE": true}
	if !validViews[view] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "View must be EXPIRED, PENDING, RECENT, DISABLED, ONLINE or OFFLINE"})
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
		View:     view,
		Now:      time.Now(),
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

func customerDuplicateErrorResponse(err error) (int, string, bool) {
	switch {
	case errors.Is(err, services.ErrCustomerMobileExists):
		return http.StatusConflict,
			"A customer with this mobile number already exists",
			true
	case errors.Is(err, services.ErrCustomerNIDExists):
		return http.StatusConflict,
			"A customer with this NID already exists",
			true
	default:
		return 0, "", false
	}
}

func parseOptionalCustomerDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse("02-01-2006", value)
	if err != nil {
		return nil, fmt.Errorf("date must use DD-MM-YYYY format")
	}

	return &parsed, nil
}

func parseCustomerProfileDates(
	dateOfBirth string,
	joiningDate string,
	nidBirthDate string,
	nidIssueDate string,
) (*time.Time, *time.Time, *time.Time, *time.Time, error) {
	dob, err := parseOptionalCustomerDate(dateOfBirth)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("date_of_birth: %w", err)
	}

	joining, err := parseOptionalCustomerDate(joiningDate)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("joining_date: %w", err)
	}

	nidBirth, err := parseOptionalCustomerDate(nidBirthDate)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("nid_birth_date: %w", err)
	}

	nidIssue, err := parseOptionalCustomerDate(nidIssueDate)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("nid_issue_date: %w", err)
	}

	return dob, joining, nidBirth, nidIssue, nil
}

func CreateCustomer(c *gin.Context) {
	var req dto.CreateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}
	if c.GetString("role") == "agent" {
		agentID := c.GetUint("agent_id")
		if agentID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Agent account is not linked"})
			return
		}
		req.AgentID = &agentID
	}
	if err := services.ValidateCustomerDistribution(req.PopID, req.AgentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.ValidateCustomerIdentity(req.Mobile, req.AltMobile, req.NID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dateOfBirth, joiningDate, nidBirthDate, nidIssueDate, err :=
		parseCustomerProfileDates(
			req.DateOfBirth,
			req.JoiningDate,
			req.NIDBirthDate,
			req.NIDIssueDate,
		)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := models.Customer{
		FullName:         strings.TrimSpace(req.FullName),
		Mobile:           strings.TrimSpace(req.Mobile),
		FatherName:       strings.TrimSpace(req.FatherName),
		MotherName:       strings.TrimSpace(req.MotherName),
		AltMobile:        strings.TrimSpace(req.AltMobile),
		Email:            strings.TrimSpace(req.Email),
		NID:              strings.TrimSpace(req.NID),
		DateOfBirth:      dateOfBirth,
		JoiningDate:      joiningDate,
		Occupation:       strings.TrimSpace(req.Occupation),
		CompanyName:      strings.TrimSpace(req.CompanyName),
		Designation:      strings.TrimSpace(req.Designation),
		NIDBirthDate:     nidBirthDate,
		NIDIssueDate:     nidIssueDate,
		NIDAddress:       strings.TrimSpace(req.NIDAddress),
		PresentAddress:   strings.TrimSpace(req.PresentAddress),
		PermanentAddress: strings.TrimSpace(req.PermanentAddress),
		TIN:              strings.TrimSpace(req.TIN),
		CustomerNote:     strings.TrimSpace(req.CustomerNote),
		Country:          strings.TrimSpace(req.Country),
		Division:         strings.TrimSpace(req.Division),
		District:         strings.TrimSpace(req.District),
		Upazila:          strings.TrimSpace(req.Upazila),
		PostOffice:       strings.TrimSpace(req.PostOffice),
		PostalCode:       strings.TrimSpace(req.PostalCode),
		RoadOrArea:       strings.TrimSpace(req.RoadOrArea),
		VillageOrHolding: strings.TrimSpace(req.VillageOrHolding),
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
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
		if status, message, handled := customerDuplicateErrorResponse(err); handled {
			c.JSON(status, gin.H{"error": message})
			return
		}

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
	if err := services.ValidateCustomerIdentity(req.Mobile, req.AltMobile, req.NID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dateOfBirth, joiningDate, nidBirthDate, nidIssueDate, err :=
		parseCustomerProfileDates(
			req.DateOfBirth,
			req.JoiningDate,
			req.NIDBirthDate,
			req.NIDIssueDate,
		)
	if err != nil {
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
	customer.DateOfBirth = dateOfBirth
	customer.JoiningDate = joiningDate
	customer.Occupation = strings.TrimSpace(req.Occupation)
	customer.CompanyName = strings.TrimSpace(req.CompanyName)
	customer.Designation = strings.TrimSpace(req.Designation)
	customer.NIDBirthDate = nidBirthDate
	customer.NIDIssueDate = nidIssueDate
	customer.NIDAddress = strings.TrimSpace(req.NIDAddress)
	customer.PresentAddress = strings.TrimSpace(req.PresentAddress)
	customer.PermanentAddress = strings.TrimSpace(req.PermanentAddress)
	customer.TIN = strings.TrimSpace(req.TIN)
	customer.CustomerNote = strings.TrimSpace(req.CustomerNote)
	customer.Country = strings.TrimSpace(req.Country)
	customer.Division = strings.TrimSpace(req.Division)
	customer.District = strings.TrimSpace(req.District)
	customer.Upazila = strings.TrimSpace(req.Upazila)
	customer.PostOffice = strings.TrimSpace(req.PostOffice)
	customer.PostalCode = strings.TrimSpace(req.PostalCode)
	customer.RoadOrArea = strings.TrimSpace(req.RoadOrArea)
	customer.VillageOrHolding = strings.TrimSpace(req.VillageOrHolding)
	customer.Latitude = req.Latitude
	customer.Longitude = req.Longitude
	customer.Union = strings.TrimSpace(req.Union)
	customer.Village = strings.TrimSpace(req.Village)
	customer.Address = strings.TrimSpace(req.Address)
	customer.BillingDay = req.BillingDay
	customer.PopID = req.PopID
	customer.AgentID = req.AgentID

	if err := services.UpdateCustomer(customer); err != nil {
		if status, message, handled := customerDuplicateErrorResponse(err); handled {
			c.JSON(status, gin.H{"error": message})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update customer",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToCustomerResponse(*customer))
}

func UpdateCustomerStatus(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		customer, err = services.UpdateCustomerStatusWithInternetAccounts(uint(id), req.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer status"})
			return
		}

		subscriptions, listErr := services.GetSubscriptionsByCustomer(uint(id))
		if listErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Customer status saved, but subscriptions could not be loaded", "customer": dto.ToCustomerResponse(*customer)})
			return
		}
		for _, subscription := range subscriptions {
			if subscription.Status == "DISCONNECTED" {
				continue
			}
			if _, syncErr := services.ReconcileSubscriptionPPPSecretWithMikroTik(subscription.ID, cfg.CredentialKey); syncErr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "Customer status saved, but MikroTik synchronization failed: " + syncErr.Error(), "customer": dto.ToCustomerResponse(*customer)})
				return
			}
		}

		c.JSON(http.StatusOK, dto.ToCustomerResponse(*customer))
	}
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
