package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func customerForReferenceRequest(
	c *gin.Context,
) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Invalid customer ID"},
		)
		return 0, false
	}

	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "Customer not found"},
		)
		return 0, false
	}

	if !canAccessCustomer(c, customer) {
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "Permission denied"},
		)
		return 0, false
	}

	return uint(id), true
}

func ListCustomerReferences(c *gin.Context) {
	customerID, ok := customerForReferenceRequest(c)
	if !ok {
		return
	}

	rows, err := services.ListCustomerReferences(customerID)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to load customer references"},
		)
		return
	}

	response := make(
		[]dto.CustomerReferenceResponse,
		0,
		len(rows),
	)

	for _, row := range rows {
		response = append(
			response,
			dto.ToCustomerReferenceResponse(row),
		)
	}

	c.JSON(http.StatusOK, gin.H{"references": response})
}

func CreateCustomerReference(c *gin.Context) {
	customerID, ok := customerForReferenceRequest(c)
	if !ok {
		return
	}

	var req dto.CustomerReferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Invalid customer reference data"},
		)
		return
	}

	row, err := services.CreateCustomerReference(
		customerID,
		services.CustomerReferenceInput{
			Name:     req.Name,
			Mobile:   req.Mobile,
			Address:  req.Address,
			Relation: req.Relation,
			Note:     req.Note,
		},
	)
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusCreated,
		dto.ToCustomerReferenceResponse(*row),
	)
}

func UpdateCustomerReference(c *gin.Context) {
	customerID, ok := customerForReferenceRequest(c)
	if !ok {
		return
	}

	referenceID, err := strconv.ParseUint(
		c.Param("reference_id"),
		10,
		64,
	)
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Invalid reference ID"},
		)
		return
	}

	row, err := services.GetCustomerReference(
		customerID,
		uint(referenceID),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": "Customer reference not found"},
			)
			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to load customer reference"},
		)
		return
	}

	var req dto.CustomerReferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Invalid customer reference data"},
		)
		return
	}

	if err := services.UpdateCustomerReference(
		row,
		services.CustomerReferenceInput{
			Name:     req.Name,
			Mobile:   req.Mobile,
			Address:  req.Address,
			Relation: req.Relation,
			Note:     req.Note,
		},
	); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerReferenceResponse(*row),
	)
}

func DeleteCustomerReference(c *gin.Context) {
	customerID, ok := customerForReferenceRequest(c)
	if !ok {
		return
	}

	referenceID, err := strconv.ParseUint(
		c.Param("reference_id"),
		10,
		64,
	)
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Invalid reference ID"},
		)
		return
	}

	row, err := services.GetCustomerReference(
		customerID,
		uint(referenceID),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": "Customer reference not found"},
			)
			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to load customer reference"},
		)
		return
	}

	if err := services.DeleteCustomerReference(row); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to delete customer reference"},
		)
		return
	}

	c.Status(http.StatusNoContent)
}
