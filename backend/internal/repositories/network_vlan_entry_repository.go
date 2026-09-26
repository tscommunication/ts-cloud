package repositories

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func ListNetworkVLANEntries(deviceType string, deviceID uint, portName string) ([]models.NetworkVLANEntry, error) {
	var rows []models.NetworkVLANEntry
	err := database.DB.
		Where("device_type = ? AND device_id = ? AND port_name = ?", deviceType, deviceID, portName).
		Order("vlan_id, id").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list network VLAN entries: %w", err)
	}
	return rows, nil
}

func GetNetworkVLANEntry(id uint) (*models.NetworkVLANEntry, error) {
	var row models.NetworkVLANEntry
	if err := database.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func CreateNetworkVLANEntry(row *models.NetworkVLANEntry) error {
	if err := database.DB.Create(row).Error; err != nil {
		return fmt.Errorf("create network VLAN entry: %w", err)
	}
	return nil
}

func UpdateNetworkVLANEntry(row *models.NetworkVLANEntry) error {
	result := database.DB.Model(&models.NetworkVLANEntry{}).
		Where("id = ?", row.ID).
		Updates(map[string]any{
			"device_type": row.DeviceType,
			"device_id":   row.DeviceID,
			"port_name":   row.PortName,
			"vlan_id":     row.VLANID,
			"vlan_name":   row.VLANName,
		})
	if result.Error != nil {
		return fmt.Errorf("update network VLAN entry: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func DeleteNetworkVLANEntry(id uint) error {
	result := database.DB.Delete(&models.NetworkVLANEntry{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete network VLAN entry: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func IsDuplicateNetworkVLANEntry(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
