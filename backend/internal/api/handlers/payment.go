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
	if c.GetString("role") == "agent" {
		allowed, checkErr := services.InvoiceBelongsToAgent(invoice.ID, c.GetUint("agent_id"))
		if checkErr != nil || !allowed {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
			return
		}
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
	actorID := c.GetUint("user_id")
	payment.CollectedByUserID = &actorID
	if agentID := c.GetUint("agent_id"); agentID > 0 {
		payment.CollectedByAgentID = &agentID
	}

	if err := services.CreatePayment(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if saved, loadErr := services.GetPaymentByID(payment.ID); loadErr == nil {
		payment = *saved
	}
	c.JSON(http.StatusCreated, dto.ToPaymentResponse(payment))
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

	var payments []models.Payment
	var err error
	if c.GetString("role") == "agent" {
		payments, err = services.GetPaymentsByAgent(c.GetUint("agent_id"))
	} else {
		payments, err = services.GetPayments()
	}
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
	if c.GetString("role") == "agent" {
		allowed, checkErr := services.PaymentBelongsToAgent(payment.ID, c.GetUint("agent_id"))
		if checkErr != nil || !allowed {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}
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

type paymentVoidRunner func(
	id uint,
	reason string,
	voidedByUserID uint,
	now time.Time,
) (*models.PaymentVoidAudit, error)

// VoidPayment godoc
//
//	@Summary		Void Payment
//	@Description	Void a payment while preserving financial history and audit trail
//	@Tags			Payment
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	int						true	"Payment ID"
//	@Param			request	body	dto.VoidPaymentRequest	true	"Void reason"
//	@Success		200		{object}	map[string]interface{}
//	@Router			/api/v1/payments/{id}/void [post]
func voidPaymentHandler(
	runner paymentVoidRunner,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(
			c.Param("id"),
			10,
			64,
		)
		if err != nil || id == 0 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Invalid payment ID"},
			)
			return
		}

		var req dto.VoidPaymentRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "void reason is required"},
			)
			return
		}

		actorID := c.GetUint("user_id")
		if actorID == 0 {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{"error": "authenticated user is required"},
			)
			return
		}

		if runner == nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "payment void runner is not configured",
				},
			)
			return
		}

		audit, err := runner(
			uint(id),
			req.Reason,
			actorID,
			time.Now(),
		)
		if err != nil {
			c.JSON(
				http.StatusConflict,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"message": "Payment voided successfully",
				"audit":   audit,
			},
		)
	}
}

func VoidPayment(c *gin.Context) {
	voidPaymentHandler(
		services.VoidPayment,
	)(c)
}
