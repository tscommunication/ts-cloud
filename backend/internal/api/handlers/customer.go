package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetCustomers(c *gin.Context) {
	customers, err := services.GetAllCustomers()
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
		"count":     len(response),
		"customers": response,
	})
}

func CreateCustomer(c *gin.Context) {
	var req dto.CreateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	// Generate Customer Code
	customerCode := "CUS-000001"

	lastCustomer, err := services.GetLastCustomer()
	if err == nil {
		customerCode = fmt.Sprintf("CUS-%06d", lastCustomer.ID+1)
	}

	customer := models.Customer{
		CustomerCode: customerCode,
		FullName:     req.FullName,
		Mobile:       req.Mobile,
		FatherName:   req.FatherName,
		MotherName:   req.MotherName,
		AltMobile:    req.AltMobile,
		Email:        req.Email,
		NID:          req.NID,
		Division:     req.Division,
		District:     req.District,
		Upazila:      req.Upazila,
		Union:        req.Union,
		Village:      req.Village,
		Address:      req.Address,
		BillingDay:   req.BillingDay,
		Status:       "ACTIVE",
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
