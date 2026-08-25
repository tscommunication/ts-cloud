package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type networkDeviceRequest struct {
	Code               string `json:"code" binding:"required"`
	Name               string `json:"name" binding:"required"`
	DeviceType         string `json:"device_type" binding:"required"`
	Vendor             string `json:"vendor" binding:"required"`
	Model              string `json:"model" binding:"required"`
	OLTType            string `json:"olt_type"`
	POPID              *uint  `json:"pop_id"`
	ManagementIP       string `json:"management_ip" binding:"required"`
	ManagementPort     int    `json:"management_port"`
	RouterIDs          []uint `json:"router_ids"`
	MonitoringProtocol string `json:"monitoring_protocol" binding:"required"`
	SNMPVersion        string `json:"snmp_version"`
	SNMPPort           int    `json:"snmp_port"`
	SNMPUsername       string `json:"snmp_username"`
	SNMPSecret         string `json:"snmp_secret"`
	PollingInterval    int    `json:"polling_interval_seconds"`
	MonitoringEnabled  bool   `json:"monitoring_enabled"`
	Remarks            string `json:"remarks"`
}

func requireAgentNetworkDeviceAccess(
	c *gin.Context,
	deviceID uint,
) bool {
	if c.GetString("role") != "agent" {
		return true
	}

	agentID := c.GetUint("agent_id")
	if agentID == 0 {
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "Agent account is not linked"},
		)
		return false
	}

	allowed, err := repositories.AgentHasNetworkDevice(
		agentID,
		deviceID,
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Failed to verify network device access",
			},
		)
		return false
	}

	if !allowed {
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "Network device access denied"},
		)
		return false
	}

	return true
}

func networkDeviceResponse(row models.NetworkDevice, credentialKey string) gin.H {
	popName := ""
	if row.POP != nil {
		popName = row.POP.Name
	}
	routerIDs := make([]uint, 0, len(row.Routers))
	routerNames := make([]string, 0, len(row.Routers))
	for _, router := range row.Routers {
		routerIDs = append(routerIDs, router.ID)
		routerNames = append(routerNames, router.Code+" — "+router.Name)
	}
	return gin.H{"id": row.ID, "code": row.Code, "name": row.Name, "device_type": row.DeviceType, "vendor": row.Vendor, "model": row.DeviceModel, "olt_type": row.OLTType, "pop_id": row.POPID, "pop_name": popName, "management_ip": row.ManagementIP, "management_port": row.ManagementPort, "router_ids": routerIDs, "router_names": routerNames, "monitoring_protocol": row.MonitoringProtocol, "snmp_version": row.SNMPVersion, "snmp_port": row.SNMPPort, "snmp_username": row.SNMPUsername, "credential_configured": row.SNMPSecretEncrypted != "", "polling_interval_seconds": row.PollingInterval, "monitoring_enabled": row.MonitoringEnabled, "monitoring_status": row.MonitoringStatus, "last_polled_at": row.LastPolledAt, "last_error": row.LastError, "remarks": row.Remarks}
}

func ListNetworkDevices(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rows []models.NetworkDevice
		var err error

		if c.GetString("role") == "agent" {
			agentID := c.GetUint("agent_id")
			if agentID == 0 {
				c.JSON(
					http.StatusForbidden,
					gin.H{"error": "Agent account is not linked"},
				)
				return
			}

			rows, err = services.ListNetworkDevicesForAgent(
				agentID,
			)
		} else {
			rows, err = services.ListNetworkDevices()
		}

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "Failed to load network devices"},
			)
			return
		}
		result := make([]gin.H, 0, len(rows))
		for _, row := range rows {
			result = append(result, networkDeviceResponse(row, cfg.CredentialKey))
		}
		c.JSON(200, gin.H{"devices": result})
	}
}

func SaveNetworkDevice(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req networkDeviceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		row := models.NetworkDevice{Code: req.Code, Name: req.Name, DeviceType: req.DeviceType, Vendor: req.Vendor, DeviceModel: req.Model, OLTType: req.OLTType, POPID: req.POPID, ManagementIP: req.ManagementIP, ManagementPort: req.ManagementPort, MonitoringProtocol: req.MonitoringProtocol, SNMPVersion: req.SNMPVersion, SNMPPort: req.SNMPPort, SNMPUsername: req.SNMPUsername, PollingInterval: req.PollingInterval, MonitoringEnabled: req.MonitoringEnabled, Remarks: req.Remarks}
		if raw := c.Param("id"); raw != "" {
			id, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid device ID"})
				return
			}
			existing, err := services.GetNetworkDevice(uint(id))
			if err != nil {
				c.JSON(404, gin.H{"error": "Device not found"})
				return
			}
			row.ID = existing.ID
			row.SNMPSecretEncrypted = existing.SNMPSecretEncrypted
			row.MonitoringStatus = existing.MonitoringStatus
			row.LastPolledAt = existing.LastPolledAt
			row.LastError = existing.LastError
		}
		if err := services.SaveNetworkDevice(&row, req.RouterIDs, req.SNMPSecret, cfg.CredentialKey); err != nil {
			c.JSON(422, gin.H{"error": err.Error()})
			return
		}
		saved, _ := services.GetNetworkDevice(row.ID)
		c.JSON(200, networkDeviceResponse(*saved, cfg.CredentialKey))
	}
}

func DeleteNetworkDevice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid device ID"})
		return
	}
	err = services.DeleteNetworkDevice(uint(id))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{"error": "Device not found"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete device"})
		return
	}
	c.Status(http.StatusNoContent)
}

func TestNetworkDeviceConnection(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid device ID"})
			return
		}
		row, err := services.TestNetworkDeviceConnection(uint(id), cfg.CredentialKey)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Device not found"})
			return
		}
		if err != nil {
			c.JSON(422, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, networkDeviceResponse(*row, cfg.CredentialKey))
	}
}
