package repositories

import (
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupOLTDashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(filepath.Join(t.TempDir(), "olt-dashboard.db")),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.POP{},
		&models.NetworkRouter{},
		&models.NetworkDevice{},
		&models.AgentNetworkDevice{},
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previous
	})

	return db
}

func TestGetOLTDashboardSummaryAndAgentScope(t *testing.T) {
	db := setupOLTDashboardTestDB(t)

	popA := models.POP{
		Code:   "POP-A",
		Name:   "POP A",
		Status: "ACTIVE",
	}
	popB := models.POP{
		Code:   "POP-B",
		Name:   "POP B",
		Status: "ACTIVE",
	}

	if err := db.Create(&popA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&popB).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	oltA := models.NetworkDevice{
		Code:               "OLT-A",
		Name:               "OLT A",
		DeviceType:         "OLT",
		Vendor:             "VSOL",
		DeviceModel:        "TEST",
		OLTType:            "EPON",
		POPID:              &popA.ID,
		ManagementIP:       "192.0.2.10",
		ManagementPort:     80,
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		PollingInterval:    300,
		MonitoringEnabled:  true,
		MonitoringStatus:   "ONLINE",
		LastPolledAt:       &now,
	}

	oltB := models.NetworkDevice{
		Code:               "OLT-B",
		Name:               "OLT B",
		DeviceType:         "OLT",
		Vendor:             "ECOM",
		DeviceModel:        "TEST",
		OLTType:            "EPON",
		POPID:              &popB.ID,
		ManagementIP:       "192.0.2.20",
		ManagementPort:     80,
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		PollingInterval:    300,
		MonitoringEnabled:  true,
		MonitoringStatus:   "OFFLINE",
		LastPolledAt:       &now,
	}

	if err := db.Create(&oltA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&oltB).Error; err != nil {
		t.Fatal(err)
	}

	onus := []models.NetworkDeviceONU{
		{
			NetworkDeviceID: oltA.ID,
			PONNo:           1,
			ONUNo:           1,
			MACAddress:      "00:00:00:00:00:01",
			OperStatus:      "UP",
			LastSeenAt:      &now,
		},
		{
			NetworkDeviceID: oltA.ID,
			PONNo:           1,
			ONUNo:           2,
			MACAddress:      "00:00:00:00:00:02",
			OperStatus:      "DOWN",
			LastSeenAt:      &now,
		},
		{
			NetworkDeviceID: oltB.ID,
			PONNo:           1,
			ONUNo:           1,
			MACAddress:      "00:00:00:00:00:03",
			OperStatus:      "UP",
			LastSeenAt:      &now,
		},
	}

	for i := range onus {
		if err := db.Create(&onus[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	rx := -18.5
	if err := db.Create(&models.NetworkDeviceONUSample{
		NetworkDeviceONUID: onus[0].ID,
		SampledAt:          now,
		RxPowerDBM:         &rx,
	}).Error; err != nil {
		t.Fatal(err)
	}

	all, err := GetOLTDashboard(0)
	if err != nil {
		t.Fatal(err)
	}

	if all.Summary.TotalOLTs != 2 ||
		all.Summary.OnlineOLTs != 1 ||
		all.Summary.OfflineOLTs != 1 {
		t.Fatalf("unexpected OLT summary: %+v", all.Summary)
	}

	if all.Summary.TotalONUs != 3 ||
		all.Summary.OnlineONUs != 2 ||
		all.Summary.OfflineONUs != 1 {
		t.Fatalf("unexpected ONU summary: %+v", all.Summary)
	}

	if all.Summary.OpticalMissing != 2 {
		t.Fatalf(
			"optical_missing=%d want=2",
			all.Summary.OpticalMissing,
		)
	}

	const agentID uint = 77

	if err := db.Create(&models.AgentNetworkDevice{
		AgentID:         agentID,
		NetworkDeviceID: oltA.ID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scoped, err := GetOLTDashboard(agentID)
	if err != nil {
		t.Fatal(err)
	}

	if scoped.Summary.TotalOLTs != 1 ||
		scoped.Summary.TotalONUs != 2 ||
		scoped.Summary.OnlineONUs != 1 ||
		scoped.Summary.OfflineONUs != 1 {
		t.Fatalf("unexpected agent scoped summary: %+v", scoped.Summary)
	}

	if len(scoped.OLTs) != 1 || scoped.OLTs[0].ID != oltA.ID {
		t.Fatalf("unexpected scoped OLT rows: %+v", scoped.OLTs)
	}
}
