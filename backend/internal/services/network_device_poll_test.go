package services

import (
	"gorm.io/gorm"

	"errors"
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func encryptedPollTestCommunity(
	t *testing.T,
	key string,
) string {
	t.Helper()

	encrypted, err := security.EncryptSecret(
		"tscloud",
		key,
	)
	if err != nil {
		t.Fatal(err)
	}

	return encrypted
}

func TestPollNetworkDeviceSNMPv2cProbeAndTelemetry(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	in := uint64(100)
	out := uint64(200)

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 7},
		ManagementIP:       "192.0.2.10",
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		SNMPSecretEncrypted: encryptedPollTestCommunity(
			t,
			key,
		),
	}

	sampledAt := time.Date(
		2026,
		time.August,
		23,
		6,
		10,
		0,
		0,
		time.UTC,
	)

	probeCalls := 0
	collectCalls := 0
	persistCalls := 0

	result, err := pollNetworkDeviceSNMPv2c(
		device,
		key,
		sampledAt,
		networkDevicePollDeps{
			probe: func(
				host string,
				port int,
				community string,
			) (string, error) {
				probeCalls++

				if host != device.ManagementIP ||
					port != 161 ||
					community != "tscloud" {
					t.Fatal("unexpected probe input")
				}

				return "ONLINE", nil
			},

			collect: func(
				cfg snmpmonitor.V2CConfig,
				gotSampledAt time.Time,
			) (*snmpmonitor.IFMIBCollection, error) {
				collectCalls++

				if cfg.Community != "tscloud" {
					t.Fatal("unexpected collector community")
				}

				if !gotSampledAt.Equal(sampledAt) {
					t.Fatal("unexpected sample time")
				}

				return &snmpmonitor.IFMIBCollection{
					SampledAt: sampledAt,
					Ports: []snmpmonitor.IFMIBPort{
						{
							IfIndex:     1,
							Name:        "eth0/0/1",
							IfType:      6,
							AdminStatus: 1,
							OperStatus:  1,
							HCInOctets:  &in,
							HCOutOctets: &out,
						},
					},
				}, nil
			},

			persist: func(
				networkDeviceID uint,
				candidates []snmpmonitor.PortPersistenceCandidate,
			) error {
				persistCalls++

				if networkDeviceID != 7 {
					t.Fatal("unexpected network device ID")
				}

				if len(candidates) != 1 {
					t.Fatalf(
						"expected 1 candidate got %d",
						len(candidates),
					)
				}

				return nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != "ONLINE" {
		t.Fatalf(
			"unexpected status %q",
			result.Status,
		)
	}

	if result.ProbeError != nil {
		t.Fatalf(
			"unexpected probe error: %v",
			result.ProbeError,
		)
	}

	if result.TelemetryError != nil {
		t.Fatalf(
			"unexpected telemetry error: %v",
			result.TelemetryError,
		)
	}

	if result.PortCount != 1 {
		t.Fatalf(
			"expected 1 port got %d",
			result.PortCount,
		)
	}

	if probeCalls != 1 ||
		collectCalls != 1 ||
		persistCalls != 1 {
		t.Fatalf(
			"unexpected calls probe=%d collect=%d persist=%d",
			probeCalls,
			collectCalls,
			persistCalls,
		)
	}
}

func TestPollNetworkDeviceSNMPv2cKeepsProbeStatusWhenTelemetryFails(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 8},
		ManagementIP:       "192.0.2.20",
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		SNMPSecretEncrypted: encryptedPollTestCommunity(
			t,
			key,
		),
	}

	result, err := pollNetworkDeviceSNMPv2c(
		device,
		key,
		time.Now(),
		networkDevicePollDeps{
			probe: func(
				string,
				int,
				string,
			) (string, error) {
				return "ONLINE", nil
			},

			collect: func(
				snmpmonitor.V2CConfig,
				time.Time,
			) (*snmpmonitor.IFMIBCollection, error) {
				return nil, errors.New(
					"collector unavailable",
				)
			},

			persist: func(
				uint,
				[]snmpmonitor.PortPersistenceCandidate,
			) error {
				t.Fatal(
					"persist must not run after collection failure",
				)
				return nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != "ONLINE" {
		t.Fatalf(
			"unexpected status %q",
			result.Status,
		)
	}

	if result.TelemetryError == nil {
		t.Fatal("expected telemetry error")
	}
}

func TestPollNetworkDeviceSNMPv2cStillAttemptsTelemetryAfterProbeError(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	in := uint64(1)
	out := uint64(2)

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 9},
		ManagementIP:       "192.0.2.30",
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		SNMPSecretEncrypted: encryptedPollTestCommunity(
			t,
			key,
		),
	}

	persisted := false

	result, err := pollNetworkDeviceSNMPv2c(
		device,
		key,
		time.Now(),
		networkDevicePollDeps{
			probe: func(
				string,
				int,
				string,
			) (string, error) {
				return "DEGRADED",
					errors.New("probe response error")
			},

			collect: func(
				_ snmpmonitor.V2CConfig,
				sampledAt time.Time,
			) (*snmpmonitor.IFMIBCollection, error) {
				return &snmpmonitor.IFMIBCollection{
					SampledAt: sampledAt,
					Ports: []snmpmonitor.IFMIBPort{
						{
							IfIndex:     1,
							HCInOctets:  &in,
							HCOutOctets: &out,
						},
					},
				}, nil
			},

			persist: func(
				uint,
				[]snmpmonitor.PortPersistenceCandidate,
			) error {
				persisted = true
				return nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != "DEGRADED" {
		t.Fatalf(
			"unexpected status %q",
			result.Status,
		)
	}

	if result.ProbeError == nil {
		t.Fatal("expected probe error")
	}

	if !persisted {
		t.Fatal(
			"telemetry was not persisted after probe error",
		)
	}
}
