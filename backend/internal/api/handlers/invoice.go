package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

// GetInvoices godoc
//
//	@Summary		Get Invoices
//	@Description	Get all invoices
//	@Tags			Invoice
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200 {array} dto.InvoiceResponse
//	@Router			/api/v1/invoices [get]
func GetInvoices(c *gin.Context) {

	var invoices []models.Invoice
	var err error
	if c.GetString("role") == "agent" {
		invoices, err = services.GetInvoicesByAgent(c.GetUint("agent_id"))
	} else {
		invoices, err = services.GetInvoices()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch invoices",
		})
		return
	}

	response := make([]dto.InvoiceResponse, 0)

	for _, invoice := range invoices {
		response = append(response, dto.ToInvoiceResponse(invoice))
	}

	c.JSON(http.StatusOK, response)
}

// GetInvoice godoc
//
//	@Summary		Get Invoice
//	@Description	Get invoice by ID
//	@Tags			Invoice
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id path int true "Invoice ID"
//	@Success		200 {object} dto.InvoiceResponse
//	@Router			/api/v1/invoices/{id} [get]
func GetInvoice(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}

	invoice, err := services.GetInvoiceByID(uint(id))
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

	c.JSON(http.StatusOK, dto.ToInvoiceResponse(*invoice))
}

// CreateInvoice godoc
//
//	@Summary		Create Invoice
//	@Description	Create new invoice
//	@Tags			Invoice
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request body dto.CreateInvoiceRequest true "Invoice"
//	@Success		201 {object} dto.InvoiceResponse
//	@Router			/api/v1/invoices [post]
func CreateInvoice(c *gin.Context) {

	var req dto.CreateInvoiceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	issueDate, dueDate, err := parseInvoiceDates(req.IssueDate, req.DueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	total := req.PackagePrice - req.Discount + req.Vat

	invoice := models.Invoice{
		SubscriptionID: req.SubscriptionID,

		BillMonth: req.BillMonth,
		BillYear:  req.BillYear,

		IssueDate: issueDate,
		DueDate:   dueDate,

		PackagePrice: req.PackagePrice,
		Discount:     req.Discount,
		Vat:          req.Vat,

		TotalAmount: total,
		PaidAmount:  0,
		DueAmount:   total,

		Status:  "UNPAID",
		Remarks: req.Remarks,
	}

	if err := services.CreateInvoice(&invoice); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Reload invoice from DB so CustomerID/PackageID and relations are fresh
	invoiceData, err := services.GetInvoiceByID(invoice.ID)
	if err != nil {
		c.JSON(http.StatusCreated, dto.ToInvoiceResponse(invoice))
		return
	}

	c.JSON(http.StatusCreated, dto.ToInvoiceResponse(*invoiceData))
}

// UpdateInvoice godoc
//
//	@Summary		Update Invoice
//	@Description	Update invoice
//	@Tags			Invoice
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id path int true "Invoice ID"
//	@Param			request body dto.CreateInvoiceRequest true "Invoice"
//	@Success		200 {object} dto.InvoiceResponse
//	@Router			/api/v1/invoices/{id} [put]
func UpdateInvoice(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}

	invoice, err := services.GetInvoiceByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Invoice not found",
		})
		return
	}

	var req dto.CreateInvoiceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	issueDate, dueDate, err := parseInvoiceDates(req.IssueDate, req.DueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoice.SubscriptionID = req.SubscriptionID
	invoice.BillMonth = req.BillMonth
	invoice.BillYear = req.BillYear
	invoice.PackagePrice = req.PackagePrice
	invoice.Discount = req.Discount
	invoice.Vat = req.Vat
	invoice.IssueDate = issueDate
	invoice.DueDate = dueDate

	invoice.TotalAmount = req.PackagePrice - req.Discount + req.Vat
	invoice.DueAmount = invoice.TotalAmount - invoice.PaidAmount
	invoice.Remarks = req.Remarks

	if err := services.UpdateInvoice(invoice); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToInvoiceResponse(*invoice))
}

// CancelInvoice godoc
//
//	@Summary		Cancel Invoice
//	@Description	Cancel an unpaid invoice while preserving history
//	@Tags			Invoice
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id path int true "Invoice ID"
//	@Success		200 {object} map[string]interface{}
//	@Router			/api/v1/invoices/{id}/cancel [post]
func CancelInvoice(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}

	invoice, err := services.GetInvoiceByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
		return
	}
	if err := services.CancelInvoice(invoice); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invoice cancelled successfully",
	})
}

func parseInvoiceDates(issueValue, dueValue string) (time.Time, time.Time, error) {
	issueDate := time.Now()
	var err error
	if issueValue != "" {
		issueDate, err = time.Parse("2006-01-02", issueValue)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid issue date")
		}
	}
	dueDate := issueDate.AddDate(0, 0, 7)
	if dueValue != "" {
		dueDate, err = time.Parse("2006-01-02", dueValue)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid due date")
		}
	}
	return issueDate, dueDate, nil
}
