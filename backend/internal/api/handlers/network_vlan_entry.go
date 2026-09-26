package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type networkVLANEntryRequest struct {
	DeviceType string `json:"device_type" binding:"required"`
	DeviceID   uint   `json:"device_id" binding:"required"`
	PortName   string `json:"port_name" binding:"required"`
	VLANID     int    `json:"vlan_id" binding:"required"`
	VLANName   string `json:"vlan_name" binding:"required"`
}

func ListNetworkVLANEntries(c *gin.Context) {
	deviceType := strings.ToUpper(strings.TrimSpace(c.Query("device_type")))
	deviceID, err := strconv.ParseUint(c.Query("device_id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}
	rows, err := services.ListNetworkVLANEntries(deviceType, uint(deviceID), c.Query("port_name"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": rows})
}

func CreateNetworkVLANEntry(c *gin.Context) {
	var request networkVLANEntryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device, port, VLAN ID and VLAN name are required"})
		return
	}
	entry := models.NetworkVLANEntry{
		DeviceType: request.DeviceType,
		DeviceID:   request.DeviceID,
		PortName:   request.PortName,
		VLANID:     request.VLANID,
		VLANName:   request.VLANName,
	}
	if err := services.SaveNetworkVLANEntry(&entry); err != nil {
		writeNetworkVLANEntryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, entry)
}

func UpdateNetworkVLANEntry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid VLAN entry ID"})
		return
	}
	var request networkVLANEntryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device, port, VLAN ID and VLAN name are required"})
		return
	}
	entry := models.NetworkVLANEntry{
		Model:      gorm.Model{ID: uint(id)},
		DeviceType: request.DeviceType,
		DeviceID:   request.DeviceID,
		PortName:   request.PortName,
		VLANID:     request.VLANID,
		VLANName:   request.VLANName,
	}
	if err := services.SaveNetworkVLANEntry(&entry); err != nil {
		writeNetworkVLANEntryError(c, err)
		return
	}
	c.JSON(http.StatusOK, entry)
}

func DeleteNetworkVLANEntry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid VLAN entry ID"})
		return
	}
	if err := services.DeleteNetworkVLANEntry(uint(id)); err != nil {
		if services.IsNetworkVLANEntryNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "VLAN entry not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete VLAN entry"})
		return
	}
	c.Status(http.StatusNoContent)
}

func writeNetworkVLANEntryError(c *gin.Context, err error) {
	if services.IsNetworkVLANEntryNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Selected device or VLAN entry was not found"})
		return
	}
	if services.IsDuplicateNetworkVLANEntry(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "That VLAN ID is already saved for this device port"})
		return
	}
	c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
}
