package services

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestNetworkDevicePollDue(t *testing.T) {
	now := time.Date(2026, 8, 22, 2, 0, 0, 0, time.UTC)
	last := now.Add(-4 * time.Minute)

	tests := []struct {
		name   string
		device *models.NetworkDevice
		want   bool
	}{
		{name: "never polled", device: &models.NetworkDevice{MonitoringEnabled: true, PollingInterval: 300}, want: true},
		{name: "not due", device: &models.NetworkDevice{MonitoringEnabled: true, PollingInterval: 300, LastPolledAt: &last}, want: false},
		{name: "due", device: &models.NetworkDevice{MonitoringEnabled: true, PollingInterval: 180, LastPolledAt: &last}, want: true},
		{name: "disabled", device: &models.NetworkDevice{MonitoringEnabled: false, PollingInterval: 30}, want: false},
		{name: "nil", device: nil, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := networkDevicePollDue(test.device, now); got != test.want {
				t.Fatalf("networkDevicePollDue() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRecordNetworkDevicePollFailureReplacesStaleOnlineState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NetworkDevice{}); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	device := models.NetworkDevice{
		Code:               "OLT-MONITOR-FAIL",
		Name:               "Monitor Failure Test",
		DeviceType:         "OLT",
		Vendor:             "TEST",
		DeviceModel:        "TEST-OLT",
		ManagementIP:       "192.0.2.250",
		MonitoringProtocol: "SNMP",
		MonitoringEnabled:  true,
		MonitoringStatus:   "ONLINE",
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	observedAt := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	if err := recordNetworkDevicePollFailure(device.ID, observedAt, errors.New("decrypt SNMP community: invalid key")); err != nil {
		t.Fatal(err)
	}

	var saved models.NetworkDevice
	if err := db.First(&saved, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.MonitoringStatus != "OFFLINE" || saved.LastPolledAt == nil || !saved.LastPolledAt.Equal(observedAt) || saved.LastError != "poll: decrypt SNMP community: invalid key" {
		t.Fatalf("unexpected failure state: %+v", saved)
	}
}
