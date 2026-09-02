package repositories

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
)

type OLTDashboardSummary struct {
	TotalOLTs      int64 `json:"total_olts" gorm:"column:total_olts"`
	OnlineOLTs     int64 `json:"online_olts" gorm:"column:online_olts"`
	OfflineOLTs    int64 `json:"offline_olts" gorm:"column:offline_olts"`
	TotalONUs      int64 `json:"total_onus" gorm:"column:total_onus"`
	OnlineONUs     int64 `json:"online_onus" gorm:"column:online_onus"`
	OfflineONUs    int64 `json:"offline_onus" gorm:"column:offline_onus"`
	OpticalMissing int64 `json:"optical_missing"`
}

type OLTDashboardOLT struct {
	ID               uint       `json:"id"`
	Code             string     `json:"code"`
	Name             string     `json:"name"`
	Vendor           string     `json:"vendor"`
	OLTType          string     `json:"olt_type"`
	POPID            *uint      `json:"pop_id"`
	POPName          string     `json:"pop_name"`
	MonitoringStatus string     `json:"monitoring_status"`
	LastPolledAt     *time.Time `json:"last_polled_at"`
	LastError        string     `json:"last_error"`
	TotalONUs        int64      `json:"total_onus" gorm:"column:total_onus"`
	OnlineONUs       int64      `json:"online_onus" gorm:"column:online_onus"`
	OfflineONUs      int64      `json:"offline_onus" gorm:"column:offline_onus"`
	OpticalMissing   int64      `json:"optical_missing"`
}

type OLTDashboardVendor struct {
	Vendor      string `json:"vendor"`
	OLTCount    int64  `json:"olt_count" gorm:"column:olt_count"`
	OnlineOLTs  int64  `json:"online_olts" gorm:"column:online_olts"`
	TotalONUs   int64  `json:"total_onus" gorm:"column:total_onus"`
	OnlineONUs  int64  `json:"online_onus" gorm:"column:online_onus"`
	OfflineONUs int64  `json:"offline_onus" gorm:"column:offline_onus"`
}

type OLTDashboardPOP struct {
	POPID       *uint  `json:"pop_id"`
	POPName     string `json:"pop_name"`
	OLTCount    int64  `json:"olt_count" gorm:"column:olt_count"`
	OnlineOLTs  int64  `json:"online_olts" gorm:"column:online_olts"`
	TotalONUs   int64  `json:"total_onus" gorm:"column:total_onus"`
	OnlineONUs  int64  `json:"online_onus" gorm:"column:online_onus"`
	OfflineONUs int64  `json:"offline_onus" gorm:"column:offline_onus"`
}

type OLTDashboard struct {
	Summary OLTDashboardSummary  `json:"summary"`
	OLTs    []OLTDashboardOLT    `json:"olts"`
	Vendors []OLTDashboardVendor `json:"vendors"`
	POPs    []OLTDashboardPOP    `json:"pops"`
}

func oltDashboardScope(agentID uint) *gorm.DB {
	query := database.DB.
		Table("network_devices AS nd").
		Where("nd.deleted_at IS NULL").
		Where("UPPER(nd.device_type) = ?", "OLT")

	if agentID > 0 {
		query = query.
			Joins(
				"JOIN agent_network_devices AS andev ON andev.network_device_id = nd.id",
			).
			Where("andev.agent_id = ?", agentID)
	}

	return query
}

func GetOLTDashboard(agentID uint) (*OLTDashboard, error) {
	result := &OLTDashboard{
		OLTs:    []OLTDashboardOLT{},
		Vendors: []OLTDashboardVendor{},
		POPs:    []OLTDashboardPOP{},
	}

	if err := oltDashboardScope(agentID).
		Joins(
			"LEFT JOIN network_device_onus AS onu ON onu.network_device_id = nd.id",
		).
		Select(`
			COUNT(DISTINCT nd.id) AS total_olts,
			COUNT(DISTINCT CASE
				WHEN UPPER(COALESCE(nd.monitoring_status, '')) = 'ONLINE'
				THEN nd.id
			END) AS online_olts,
			COUNT(DISTINCT CASE
				WHEN UPPER(COALESCE(nd.monitoring_status, '')) <> 'ONLINE'
				THEN nd.id
			END) AS offline_olts,
			COUNT(onu.id) AS total_onus,
			COALESCE(SUM(CASE
				WHEN UPPER(COALESCE(onu.oper_status, '')) = 'UP'
				THEN 1 ELSE 0
			END), 0) AS online_onus,
			COALESCE(SUM(CASE
				WHEN onu.id IS NOT NULL
				 AND UPPER(COALESCE(onu.oper_status, '')) <> 'UP'
				THEN 1 ELSE 0
			END), 0) AS offline_onus,
			COALESCE(SUM(CASE
				WHEN onu.id IS NOT NULL
				 AND NOT EXISTS (
					SELECT 1
					FROM network_device_onu_samples AS sample
					WHERE sample.network_device_onu_id = onu.id
					  AND sample.rx_power_dbm IS NOT NULL
				 )
				THEN 1 ELSE 0
			END), 0) AS optical_missing
		`).
		Scan(&result.Summary).Error; err != nil {
		return nil, fmt.Errorf("load OLT dashboard summary: %w", err)
	}

	if err := oltDashboardScope(agentID).
		Joins("LEFT JOIN pops AS pop ON pop.id = nd.pop_id").
		Joins(
			"LEFT JOIN network_device_onus AS onu ON onu.network_device_id = nd.id",
		).
		Select(`
			nd.id,
			nd.code,
			nd.name,
			nd.vendor,
			nd.olt_type,
			nd.pop_id,
			COALESCE(pop.name, '') AS pop_name,
			nd.monitoring_status,
			nd.last_polled_at,
			nd.last_error,
			COUNT(onu.id) AS total_onus,
			COALESCE(SUM(CASE
				WHEN UPPER(COALESCE(onu.oper_status, '')) = 'UP'
				THEN 1 ELSE 0
			END), 0) AS online_onus,
			COALESCE(SUM(CASE
				WHEN onu.id IS NOT NULL
				 AND UPPER(COALESCE(onu.oper_status, '')) <> 'UP'
				THEN 1 ELSE 0
			END), 0) AS offline_onus,
			COALESCE(SUM(CASE
				WHEN onu.id IS NOT NULL
				 AND NOT EXISTS (
					SELECT 1
					FROM network_device_onu_samples AS sample
					WHERE sample.network_device_onu_id = onu.id
					  AND sample.rx_power_dbm IS NOT NULL
				 )
				THEN 1 ELSE 0
			END), 0) AS optical_missing
		`).
		Group(`
			nd.id,
			nd.code,
			nd.name,
			nd.vendor,
			nd.olt_type,
			nd.pop_id,
			pop.name,
			nd.monitoring_status,
			nd.last_polled_at,
			nd.last_error
		`).
		Order("nd.code ASC").
		Scan(&result.OLTs).Error; err != nil {
		return nil, fmt.Errorf("load OLT dashboard rows: %w", err)
	}

	if err := oltDashboardScope(agentID).
		Joins(
			"LEFT JOIN network_device_onus AS onu ON onu.network_device_id = nd.id",
		).
		Select(`
			nd.vendor,
			COUNT(DISTINCT nd.id) AS olt_count,
			COUNT(DISTINCT CASE
				WHEN UPPER(COALESCE(nd.monitoring_status, '')) = 'ONLINE'
				THEN nd.id
			END) AS online_olts,
			COUNT(onu.id) AS total_onus,
			COALESCE(SUM(CASE
				WHEN UPPER(COALESCE(onu.oper_status, '')) = 'UP'
				THEN 1 ELSE 0
			END), 0) AS online_onus,
			COALESCE(SUM(CASE
				WHEN onu.id IS NOT NULL
				 AND UPPER(COALESCE(onu.oper_status, '')) <> 'UP'
				THEN 1 ELSE 0
			END), 0) AS offline_onus
		`).
		Group("nd.vendor").
		Order("nd.vendor ASC").
		Scan(&result.Vendors).Error; err != nil {
		return nil, fmt.Errorf("load OLT dashboard vendor summary: %w", err)
	}

	if err := oltDashboardScope(agentID).
		Joins("LEFT JOIN pops AS pop ON pop.id = nd.pop_id").
		Joins(
			"LEFT JOIN network_device_onus AS onu ON onu.network_device_id = nd.id",
		).
		Select(`
			nd.pop_id,
			COALESCE(pop.name, 'Unassigned') AS pop_name,
			COUNT(DISTINCT nd.id) AS olt_count,
			COUNT(DISTINCT CASE
				WHEN UPPER(COALESCE(nd.monitoring_status, '')) = 'ONLINE'
				THEN nd.id
			END) AS online_olts,
			COUNT(onu.id) AS total_onus,
			COALESCE(SUM(CASE
				WHEN UPPER(COALESCE(onu.oper_status, '')) = 'UP'
				THEN 1 ELSE 0
			END), 0) AS online_onus,
			COALESCE(SUM(CASE
				WHEN onu.id IS NOT NULL
				 AND UPPER(COALESCE(onu.oper_status, '')) <> 'UP'
				THEN 1 ELSE 0
			END), 0) AS offline_onus
		`).
		Group("nd.pop_id, pop.name").
		Order("pop_name ASC").
		Scan(&result.POPs).Error; err != nil {
		return nil, fmt.Errorf("load OLT dashboard POP summary: %w", err)
	}

	return result, nil
}
