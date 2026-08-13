package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

// CreatePayment godoc
//
//	@Summary		Create Payment
//	@Description	Create new payment
//	@Tags			Payment
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreatePaymentRequest	true	"Payment"
//	@Success		201		{object}	dto.PaymentResponse
//	@Router			/api/v1/payments [post]
func CreatePayment(c *gin.Context) {

	var req dto.CreatePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	invoice, err := services.GetInvoiceByID(req.InvoiceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Invoice not found",
		})
		return
	}

	paymentDate := time.Now()

	if req.PaymentDate != "" {
		t, err := time.Parse("2006-01-02", req.PaymentDate)
		if err == nil {
			paymentDate = t
		}
	}

	payment := models.Payment{
		ReceiptNo:      "",
		InvoiceID:      invoice.ID,
		CustomerID:     invoice.CustomerID,
		SubscriptionID: invoice.SubscriptionID,

		PaymentDate: paymentDate,

		Amount: req.Amount,

		Method:        req.Method,
		TransactionID: req.TransactionID,
		Reference:     req.Reference,

		Status:  "SUCCESS",
		Remarks: req.Remarks,
	}

	if err := services.CreatePayment(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(
		http.StatusCreated,
		dto.ToPaymentResponse(payment),
	)
}

// GetPayments godoc
//
//	@Summary		Get Payment List
//	@Description	Get all payments
//	@Tags			Payment
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}	dto.PaymentResponse
//	@Router			/api/v1/payments [get]
func GetPayments(c *gin.Context) {

	payments, err := services.GetPayments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch payments",
		})
		return
	}

	response := make([]dto.PaymentResponse, 0)

	for _, payment := range payments {
		response = append(
			response,
			dto.ToPaymentResponse(payment),
		)
	}

	c.JSON(http.StatusOK, response)
}

// GetPayment godoc
//
//	@Summary		Get Payment
//	@Description	Get payment by ID
//	@Tags			Payment
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"Payment ID"
//	@Success		200	{object}	dto.PaymentResponse
//	@Router			/api/v1/payments/{id} [get]
func GetPayment(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}

	payment, err := services.GetPaymentByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Payment not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToPaymentResponse(*payment))
}

// UpdatePayment godoc
//
//	@Summary		Update Payment
//	@Description	Update payment
//	@Tags			Payment
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	int						true	"Payment ID"
//	@Param			request	body	dto.CreatePaymentRequest	true	"Payment"
//	@Success		200		{object}	dto.PaymentResponse
//	@Router			/api/v1/payments/{id} [put]
func UpdatePayment(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}

	payment, err := services.GetPaymentByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Payment not found",
		})
		return
	}

	var req dto.CreatePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	payment.Amount = req.Amount
	payment.Method = req.Method
	payment.TransactionID = req.TransactionID
	payment.Reference = req.Reference
	payment.Remarks = req.Remarks

	if req.PaymentDate != "" {
		if t, err := time.Parse("2006-01-02", req.PaymentDate); err == nil {
			payment.PaymentDate = t
		}
	}

	if err := services.UpdatePayment(payment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update payment",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToPaymentResponse(*payment))
}

// VoidPayment godoc
//
//	@Summary		Void Payment
//	@Description	Void a payment while preserving financial history
//	@Tags			Payment
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path	int	true	"Payment ID"
//	@Success		200	{object}	map[string]interface{}
//	@Router			/api/v1/payments/{id}/void [post]
func VoidPayment(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}

	if err := services.VoidPayment(uint(id)); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment voided successfully",
	})
}
