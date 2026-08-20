package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetCustomerPortalMe(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer account is not linked",
		})
		return
	}

	customer, err := services.GetCustomerByID(customerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Customer not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load customer",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToCustomerPortalMeResponse(customer))
}

func GetCustomerPortalSubscription(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer account is not linked",
		})
		return
	}

	subscriptions, err := services.GetSubscriptionsByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load subscriptions",
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerPortalSubscriptionResponses(subscriptions),
	)
}

func GetCustomerPortalInvoices(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer account is not linked",
		})
		return
	}

	invoices, err := services.GetInvoicesByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load invoices",
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerPortalInvoiceResponses(invoices),
	)
}

func GetCustomerPortalPayments(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer account is not linked",
		})
		return
	}

	payments, err := services.GetPaymentsByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load payments",
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerPortalPaymentResponses(payments),
	)
}
