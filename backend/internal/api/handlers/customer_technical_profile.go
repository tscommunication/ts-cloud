package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetCustomerTechnicalProfile(c *gin.Context) {
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

	profile, err := services.GetCustomerTechnicalProfile(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, nil)
			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to load customer technical profile"},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerTechnicalProfileResponse(*profile),
	)
}

func UpdateCustomerTechnicalProfile(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Invalid customer ID"},
			)
			return
		}

		customer, err := services.GetCustomerByID(uint(id))
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": "Customer not found"},
			)
			return
		}

		if !canAccessCustomer(c, customer) {
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "Permission denied"},
			)
			return
		}

		var req dto.CustomerTechnicalProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Invalid technical profile data"},
			)
			return
		}

		profile, err := services.SaveCustomerTechnicalProfile(
			uint(id),
			services.CustomerTechnicalProfileInput{
				ONUMAC:      req.ONUMAC,
				OLTPON:      req.OLTPON,
				OLTSlot:     req.OLTSlot,
				OLTPort:     req.OLTPort,
				ONUType:     req.ONUType,
				ONUModel:    req.ONUModel,
				ONUIP:       req.ONUIP,
				ONUPassword: req.ONUPassword,
				ONUSerial:   req.ONUSerial,
				ONUSN:       req.ONUSN,

				RouterBrand:    req.RouterBrand,
				RouterModel:    req.RouterModel,
				RouterIP:       req.RouterIP,
				RouterPassword: req.RouterPassword,

				CableType:   req.CableType,
				CableLength: req.CableLength,

				MediaConverterMAC:      req.MediaConverterMAC,
				MediaConverterIP:       req.MediaConverterIP,
				MediaConverterPassword: req.MediaConverterPassword,

				SwitchModel:    req.SwitchModel,
				SwitchPort:     req.SwitchPort,
				SwitchIP:       req.SwitchIP,
				SwitchPassword: req.SwitchPassword,

				AdditionalNote: req.AdditionalNote,
			},
			cfg.RouterCredentialKey,
		)
		if err != nil {
			c.JSON(
				http.StatusUnprocessableEntity,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.JSON(
			http.StatusOK,
			dto.ToCustomerTechnicalProfileResponse(*profile),
		)
	}
}
