package services

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func ListNetworkDevices() ([]models.NetworkDevice, error) {
	var rows []models.NetworkDevice
	err := database.DB.Preload("POP").Preload("Routers").Order("device_type, vendor, code").Find(&rows).Error
	return rows, err
}

func ListNetworkDevicesForAgent(agentID uint) ([]models.NetworkDevice, error) {
	var rows []models.NetworkDevice

	err := database.DB.
		Model(&models.NetworkDevice{}).
		Preload("POP").
		Preload("Routers").
		Joins(
			"JOIN agent_network_devices ON agent_network_devices.network_device_id = network_devices.id",
		).
		Where("agent_network_devices.agent_id = ?", agentID).
		Order("network_devices.device_type, network_devices.vendor, network_devices.code").
		Find(&rows).Error

	return rows, err
}

func GetNetworkDevice(id uint) (*models.NetworkDevice, error) {
	var row models.NetworkDevice
	if err := database.DB.Preload("POP").Preload("Routers").First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func SaveNetworkDevice(row *models.NetworkDevice, routerIDs []uint, secret, managementSecret, key string) error {
	row.Code = strings.ToUpper(strings.TrimSpace(row.Code))
	row.Name = strings.TrimSpace(row.Name)
	row.DeviceType = strings.ToUpper(strings.TrimSpace(row.DeviceType))
	row.Vendor = strings.ToUpper(strings.TrimSpace(row.Vendor))
	row.DeviceModel = strings.TrimSpace(row.DeviceModel)
	row.OLTType = strings.ToUpper(strings.TrimSpace(row.OLTType))
	row.ManagementIP = strings.TrimSpace(row.ManagementIP)
	row.MonitoringProtocol = strings.ToUpper(strings.TrimSpace(row.MonitoringProtocol))
	row.SNMPVersion = strings.ToUpper(strings.TrimSpace(row.SNMPVersion))
	row.ManagementUsername = strings.TrimSpace(row.ManagementUsername)
	row.SNMPUsername = strings.TrimSpace(row.SNMPUsername)
	if row.Code == "" || row.Name == "" || row.DeviceModel == "" || row.ManagementIP == "" {
		return errors.New("code, name, model and management IP are required")
	}
	if row.DeviceType != "OLT" && row.DeviceType != "SWITCH" && row.DeviceType != "MIKROTIK" {
		return errors.New("device type must be OLT, SWITCH or MIKROTIK")
	}
	if row.Vendor == "" {
		return errors.New("vendor is required")
	}
	if net.ParseIP(row.ManagementIP) == nil {
		if parsed, err := url.Parse("//" + row.ManagementIP); err != nil || parsed.Hostname() != row.ManagementIP || strings.ContainsAny(row.ManagementIP, " /:@") {
			return errors.New("management IP must be a valid IP address or hostname")
		}
	}
	if row.MonitoringProtocol != "SNMP" && row.MonitoringProtocol != "MIKROTIK_API" {
		return errors.New("monitoring protocol must be SNMP or MIKROTIK_API")
	}
	if row.MonitoringProtocol == "SNMP" {
		if row.SNMPVersion != "V2C" && row.SNMPVersion != "V3" {
			return errors.New("SNMP version must be V2C or V3")
		}
		if row.SNMPPort < 1 || row.SNMPPort > 65535 {
			return errors.New("SNMP port must be between 1 and 65535")
		}
	}
	if row.PollingInterval < 30 || row.PollingInterval > 86400 {
		return errors.New("polling interval must be between 30 and 86400 seconds")
	}
	if row.ManagementPort < 0 || row.ManagementPort > 65535 {
		return errors.New("management port must be between 0 and 65535")
	}
	if row.DeviceType == "OLT" {
		if row.OLTType == "" {
			return errors.New("OLT type is required")
		}
		if row.ManagementPort == 0 {
			return errors.New("management port is required for OLT devices")
		}
	}
	if row.POPID != nil {
		if _, err := repositories.GetPOP(*row.POPID); err != nil {
			return errors.New("POP not found")
		}
	}
	if strings.TrimSpace(managementSecret) != "" {
		encrypted, err := security.EncryptSecret(managementSecret, key)
		if err != nil {
			return err
		}
		row.ManagementSecretEncrypted = encrypted
	}

	if strings.TrimSpace(secret) != "" {
		encrypted, err := security.EncryptSecret(secret, key)
		if err != nil {
			return err
		}
		row.SNMPSecretEncrypted = encrypted
	}
	if row.ID == 0 && row.MonitoringProtocol == "SNMP" && row.SNMPSecretEncrypted == "" {
		return errors.New("SNMP credential is required")
	}
	if row.MonitoringStatus == "" {
		row.MonitoringStatus = "UNKNOWN"
	}
	uniqueRouterIDs := make([]uint, 0, len(routerIDs))
	seenRouterIDs := make(map[uint]struct{}, len(routerIDs))
	for _, id := range routerIDs {
		if id == 0 {
			return errors.New("invalid MikroTik router assignment")
		}
		if _, exists := seenRouterIDs[id]; !exists {
			seenRouterIDs[id] = struct{}{}
			uniqueRouterIDs = append(uniqueRouterIDs, id)
		}
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if row.ID == 0 {
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		} else if err := tx.Save(row).Error; err != nil {
			return err
		}
		var routers []models.NetworkRouter
		if len(uniqueRouterIDs) > 0 {
			if err := tx.Where("id IN ?", uniqueRouterIDs).Find(&routers).Error; err != nil {
				return err
			}
			if len(routers) != len(uniqueRouterIDs) {
				return errors.New("one or more MikroTik routers were not found")
			}
		}
		return tx.Model(row).Association("Routers").Replace(routers)
	})
}

func DeleteNetworkDevice(id uint) error {
	result := database.DB.Delete(&models.NetworkDevice{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func TestNetworkDeviceConnection(id uint, key string) (*models.NetworkDevice, error) {
	row, err := GetNetworkDevice(id)
	if err != nil {
		return nil, err
	}
	if row.MonitoringProtocol != "SNMP" {
		return nil, errors.New("connection testing currently supports SNMP devices")
	}
	if row.SNMPVersion != "V2C" {
		return nil, errors.New("SNMP v3 testing requires the v3 security profile")
	}
	community, err := security.DecryptSecret(row.SNMPSecretEncrypted, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt SNMP community: %w", err)
	}

	status, probeErr := probeSNMPv2c(row.ManagementIP, row.SNMPPort, community)

	now := time.Now()
	row.LastPolledAt = &now
	row.LastError = ""
	row.MonitoringStatus = status

	if probeErr != nil {
		row.LastError = probeErr.Error()
	}

	if err := database.DB.Model(&models.NetworkDevice{}).Where("id = ?", row.ID).Updates(map[string]any{
		"monitoring_status": row.MonitoringStatus,
		"last_polled_at":    row.LastPolledAt,
		"last_error":        row.LastError,
	}).Error; err != nil {
		return nil, err
	}

	return GetNetworkDevice(id)
}

func probeSNMPv2c(host string, port int, community string) (string, error) {
	clientConfig := snmpmonitor.V2CConfig{
		Host:      host,
		Port:      uint16(port),
		Community: community,
		Timeout:   3 * time.Second,
		Retries:   0,
	}

	allowlistedOIDs := []string{
		snmpmonitor.SysDescrOID,
		snmpmonitor.SysObjectIDOID,
		snmpmonitor.SysNameOID,
	}

	var probeErrors []error
	validResponse := false

	for _, oid := range allowlistedOIDs {
		client, err := snmpmonitor.NewV2CClient(clientConfig)
		if err != nil {
			return "OFFLINE", err
		}

		if _, err := snmpmonitor.GetOne(client, oid); err != nil {
			probeErrors = append(probeErrors, err)
			continue
		}

		validResponse = true
		break
	}

	status := classifySNMPProbeOutcome(validResponse, probeErrors)

	if validResponse {
		return status, nil
	}

	if len(probeErrors) == 0 {
		return status, errors.New("SNMP probe returned no usable response")
	}

	return status, errors.Join(probeErrors...)
}

func classifySNMPProbeOutcome(validResponse bool, probeErrors []error) string {
	if validResponse {
		return "ONLINE"
	}

	for _, err := range probeErrors {
		if snmpmonitor.IsResponseError(err) {
			return "DEGRADED"
		}
	}

	for _, err := range probeErrors {
		if snmpmonitor.IsTransportError(err) {
			return "OFFLINE"
		}
	}

	return "OFFLINE"
}
