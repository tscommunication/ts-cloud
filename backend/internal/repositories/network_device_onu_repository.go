package repositories

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func UpsertNetworkDeviceONUTx(
	tx *gorm.DB,
	row *models.NetworkDeviceONU,
) error {
	if tx == nil {
		return errors.New("database transaction is required")
	}

	if row == nil {
		return errors.New("network device ONU is required")
	}

	if row.NetworkDeviceID == 0 {
		return errors.New("network device ID is required")
	}

	if row.PONNo <= 0 {
		return fmt.Errorf(
			"invalid PON number %d",
			row.PONNo,
		)
	}

	if row.ONUNo <= 0 {
		return fmt.Errorf(
			"invalid ONU number %d",
			row.ONUNo,
		)
	}

	assignments := map[string]interface{}{
		"if_index":             row.IfIndex,
		"mac_address":          row.MACAddress,
		"serial_number":        row.SerialNumber,
		"model":                row.Model,
		"capability":           row.Capability,
		"description":          row.Description,
		"oper_status":          row.OperStatus,
		"distance_m":           row.DistanceM,
		"last_registered_at":   row.LastRegisteredAt,
		"last_deregistered_at": row.LastDeregisteredAt,
		"uptime_seconds":       row.UptimeSeconds,
		"last_seen_at":         row.LastSeenAt,
		"updated_at":           row.UpdatedAt,
	}

	if err := tx.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "network_device_id"},
				{Name: "pon_no"},
				{Name: "onu_no"},
			},
			DoUpdates: clause.Assignments(
				assignments,
			),
		},
	).Create(row).Error; err != nil {
		return fmt.Errorf(
			"upsert network device ONU: %w",
			err,
		)
	}

	var saved models.NetworkDeviceONU

	if err := tx.Where(
		"network_device_id = ? AND pon_no = ? AND onu_no = ?",
		row.NetworkDeviceID,
		row.PONNo,
		row.ONUNo,
	).First(&saved).Error; err != nil {
		return fmt.Errorf(
			"reload network device ONU: %w",
			err,
		)
	}

	*row = saved

	return nil
}

func LatestNetworkDeviceONUSampleTx(
	tx *gorm.DB,
	onuID uint,
) (*models.NetworkDeviceONUSample, error) {
	if tx == nil {
		return nil, errors.New(
			"database transaction is required",
		)
	}

	if onuID == 0 {
		return nil, errors.New(
			"network device ONU ID is required",
		)
	}

	var row models.NetworkDeviceONUSample

	err := tx.Where(
		"network_device_onu_id = ?",
		onuID,
	).Order(
		"sampled_at DESC, id DESC",
	).First(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"load latest network device ONU sample: %w",
			err,
		)
	}

	return &row, nil
}

func CreateNetworkDeviceONUSampleTx(
	tx *gorm.DB,
	row *models.NetworkDeviceONUSample,
) error {
	if tx == nil {
		return errors.New(
			"database transaction is required",
		)
	}

	if row == nil {
		return errors.New(
			"network device ONU sample is required",
		)
	}

	if row.NetworkDeviceONUID == 0 {
		return errors.New(
			"network device ONU ID is required",
		)
	}

	if row.SampledAt.IsZero() {
		return errors.New(
			"network device ONU sample time is required",
		)
	}

	if err := tx.Create(row).Error; err != nil {
		return fmt.Errorf(
			"create network device ONU sample: %w",
			err,
		)
	}

	return nil
}

func ListNetworkDeviceONUs(
	deviceID uint,
) ([]models.NetworkDeviceONU, error) {
	if deviceID == 0 {
		return nil, errors.New(
			"network device ID is required",
		)
	}

	var rows []models.NetworkDeviceONU

	if err := database.DB.Where(
		"network_device_id = ?",
		deviceID,
	).Order(
		"pon_no ASC, onu_no ASC, id ASC",
	).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf(
			"list network device ONUs: %w",
			err,
		)
	}

	return rows, nil
}

func LatestNetworkDeviceONUSample(
	onuID uint,
) (*models.NetworkDeviceONUSample, error) {
	return LatestNetworkDeviceONUSampleTx(
		database.DB,
		onuID,
	)
}

func UpsertNetworkDeviceONUTelemetryTx(
	tx *gorm.DB,
	row *models.NetworkDeviceONU,
) error {
	if tx == nil {
		return errors.New("database transaction is required")
	}

	if row == nil {
		return errors.New("network device ONU is required")
	}

	if row.NetworkDeviceID == 0 {
		return errors.New("network device ID is required")
	}

	if row.PONNo <= 0 {
		return fmt.Errorf(
			"invalid PON number %d",
			row.PONNo,
		)
	}

	if row.ONUNo <= 0 {
		return fmt.Errorf(
			"invalid ONU number %d",
			row.ONUNo,
		)
	}

	assignments := map[string]interface{}{
		"last_seen_at": row.LastSeenAt,
		"updated_at":   row.UpdatedAt,
	}

	if row.IfIndex != nil {
		assignments["if_index"] = row.IfIndex
	}

	if strings.TrimSpace(row.Description) != "" {
		assignments["description"] = row.Description
	}

	status := strings.ToUpper(
		strings.TrimSpace(row.OperStatus),
	)

	if status != "" && status != "UNKNOWN" {
		assignments["oper_status"] = status
	}

	if err := tx.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "network_device_id"},
				{Name: "pon_no"},
				{Name: "onu_no"},
			},
			DoUpdates: clause.Assignments(
				assignments,
			),
		},
	).Create(row).Error; err != nil {
		return fmt.Errorf(
			"upsert network device ONU telemetry: %w",
			err,
		)
	}

	var saved models.NetworkDeviceONU

	if err := tx.Where(
		"network_device_id = ? AND pon_no = ? AND onu_no = ?",
		row.NetworkDeviceID,
		row.PONNo,
		row.ONUNo,
	).First(&saved).Error; err != nil {
		return fmt.Errorf(
			"reload network device ONU telemetry: %w",
			err,
		)
	}

	*row = saved

	return nil
}
