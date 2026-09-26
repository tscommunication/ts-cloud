package services

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func ListNetworkVLANEntries(deviceType string, deviceID uint, portName string) ([]models.NetworkVLANEntry, error) {
	entry := models.NetworkVLANEntry{
		DeviceType: deviceType,
		DeviceID:   deviceID,
		PortName:   portName,
	}
	if err := validateNetworkVLANEntry(&entry); err != nil {
		return nil, err
	}
	if err := validateVLANEntryDevice(entry.DeviceType, entry.DeviceID); err != nil {
		return nil, err
	}
	return repositories.ListNetworkVLANEntries(entry.DeviceType, entry.DeviceID, entry.PortName)
}

func SaveNetworkVLANEntry(entry *models.NetworkVLANEntry) error {
	if entry == nil {
		return errors.New("VLAN entry is required")
	}
	if err := validateNetworkVLANEntry(entry); err != nil {
		return err
	}
	if err := validateVLANEntryDevice(entry.DeviceType, entry.DeviceID); err != nil {
		return err
	}
	if entry.ID == 0 {
		return repositories.CreateNetworkVLANEntry(entry)
	}
	if err := repositories.UpdateNetworkVLANEntry(entry); err != nil {
		return err
	}
	return nil
}

func DeleteNetworkVLANEntry(id uint) error {
	if id == 0 {
		return errors.New("VLAN entry ID is required")
	}
	return repositories.DeleteNetworkVLANEntry(id)
}

func validateNetworkVLANEntry(entry *models.NetworkVLANEntry) error {
	entry.DeviceType = strings.ToUpper(strings.TrimSpace(entry.DeviceType))
	entry.PortName = strings.TrimSpace(entry.PortName)
	entry.VLANName = strings.TrimSpace(entry.VLANName)
	if entry.DeviceType != "ROUTER" && entry.DeviceType != "SWITCH" && entry.DeviceType != "OLT" {
		return errors.New("device type must be ROUTER, SWITCH or OLT")
	}
	if entry.DeviceID == 0 || entry.PortName == "" {
		return errors.New("device ID and port name are required")
	}
	if len(entry.PortName) > 120 {
		return errors.New("port name must not exceed 120 characters")
	}
	if entry.VLANID < 1 || entry.VLANID > 4094 {
		return errors.New("VLAN ID must be between 1 and 4094")
	}
	if entry.VLANName == "" || len(entry.VLANName) > 120 {
		return errors.New("VLAN name is required and must not exceed 120 characters")
	}
	return nil
}

func validateVLANEntryDevice(deviceType string, deviceID uint) error {
	switch deviceType {
	case "ROUTER":
		if _, err := GetNetworkRouter(deviceID); err != nil {
			return fmt.Errorf("router not found: %w", err)
		}
	case "SWITCH", "OLT":
		device, err := GetNetworkDevice(deviceID)
		if err != nil {
			return fmt.Errorf("network device not found: %w", err)
		}
		if device.DeviceType != deviceType {
			return fmt.Errorf("selected device is not a %s", strings.ToLower(deviceType))
		}
	default:
		return errors.New("unsupported VLAN device type")
	}
	return nil
}

func IsNetworkVLANEntryNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func IsDuplicateNetworkVLANEntry(err error) bool {
	return repositories.IsDuplicateNetworkVLANEntry(err)
}
