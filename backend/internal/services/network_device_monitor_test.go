package services

import (
	"testing"
	"time"

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
