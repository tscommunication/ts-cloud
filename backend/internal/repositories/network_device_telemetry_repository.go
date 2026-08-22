package repositories

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tscommunication/ts-cloud/internal/models"
)

func UpsertNetworkDevicePortTx(
	tx *gorm.DB,
	row *models.NetworkDevicePort,
) error {
	if tx == nil {
		return errors.New("database transaction is required")
	}

	if row == nil {
		return errors.New("network device port is required")
	}

	if row.NetworkDeviceID == 0 {
		return errors.New("network device ID is required")
	}

	row.PortKey = strings.TrimSpace(row.PortKey)

	if row.PortKey == "" {
		return errors.New("network device port key is required")
	}

	assignments := map[string]interface{}{
		"if_index":        row.IfIndex,
		"vendor_port_ref": row.VendorPortRef,
		"name":            row.Name,
		"description":     row.Description,
		"port_type":       row.PortType,
		"admin_status":    row.AdminStatus,
		"oper_status":     row.OperStatus,
		"speed_mbps":      row.SpeedMbps,
		"mac_address":     row.MACAddress,
		"last_change_at":  row.LastChangeAt,
		"last_seen_at":    row.LastSeenAt,
		"updated_at":      row.UpdatedAt,
	}

	if err := tx.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "network_device_id"},
				{Name: "port_key"},
			},
			DoUpdates: clause.Assignments(assignments),
		},
	).Create(row).Error; err != nil {
		return fmt.Errorf(
			"upsert network device port: %w",
			err,
		)
	}

	var saved models.NetworkDevicePort

	if err := tx.Where(
		"network_device_id = ? AND port_key = ?",
		row.NetworkDeviceID,
		row.PortKey,
	).First(&saved).Error; err != nil {
		return fmt.Errorf(
			"reload network device port: %w",
			err,
		)
	}

	*row = saved

	return nil
}

func LatestNetworkDevicePortSampleTx(
	tx *gorm.DB,
	portID uint,
) (*models.NetworkDevicePortSample, error) {
	if tx == nil {
		return nil, errors.New(
			"database transaction is required",
		)
	}

	if portID == 0 {
		return nil, errors.New(
			"network device port ID is required",
		)
	}

	var row models.NetworkDevicePortSample

	err := tx.Where(
		"network_device_port_id = ?",
		portID,
	).Order(
		"sampled_at DESC, id DESC",
	).First(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"load latest network device port sample: %w",
			err,
		)
	}

	return &row, nil
}

func CreateNetworkDevicePortSampleTx(
	tx *gorm.DB,
	row *models.NetworkDevicePortSample,
) error {
	if tx == nil {
		return errors.New("database transaction is required")
	}

	if row == nil {
		return errors.New(
			"network device port sample is required",
		)
	}

	if row.NetworkDevicePortID == 0 {
		return errors.New(
			"network device port ID is required",
		)
	}

	if row.SampledAt.IsZero() {
		return errors.New(
			"network device port sample time is required",
		)
	}

	if err := tx.Create(row).Error; err != nil {
		return fmt.Errorf(
			"create network device port sample: %w",
			err,
		)
	}

	return nil
}
