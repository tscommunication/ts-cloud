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

func TestNetworkDevicePollResultKeepsONUErrorSeparate(
	t *testing.T,
) {
	result := networkDevicePollResult{
		Status:         "ONLINE",
		TelemetryError: nil,
		ONUError:       errors.New("VSOL optical unavailable"),
		PortCount:      172,
		ONUCount:       0,
		ONUAdapter:     "VSOL",
	}

	if result.Status != "ONLINE" {
		t.Fatalf(
			"status=%q want=ONLINE",
			result.Status,
		)
	}

	if result.TelemetryError != nil {
		t.Fatalf(
			"generic telemetry error must remain nil: %v",
			result.TelemetryError,
		)
	}

	if result.ONUError == nil {
		t.Fatal("expected separate ONU error")
	}

	if result.PortCount != 172 {
		t.Fatalf(
			"port count=%d want=172",
			result.PortCount,
		)
	}

	if result.ONUAdapter != "VSOL" {
		t.Fatalf(
			"ONU adapter=%q want=VSOL",
			result.ONUAdapter,
		)
	}
}

type fakeONUVendorAdapter struct {
	name string

	collectOptical func(
		cfg snmpmonitor.V2CConfig,
		sampledAt time.Time,
	) (*snmpmonitor.ONUOpticalCollection, error)

	buildCandidates func(
		ifmib *snmpmonitor.IFMIBCollection,
		optical *snmpmonitor.ONUOpticalCollection,
	) ([]snmpmonitor.ONUPersistenceCandidate, error)
}

func (adapter fakeONUVendorAdapter) Name() string {
	return adapter.name
}

func (adapter fakeONUVendorAdapter) Matches(
	string,
	string,
) bool {
	return true
}

func (adapter fakeONUVendorAdapter) CollectOptical(
	cfg snmpmonitor.V2CConfig,
	sampledAt time.Time,
) (*snmpmonitor.ONUOpticalCollection, error) {
	if adapter.collectOptical == nil {
		return nil, errors.New(
			"fake optical collector is not configured",
		)
	}

	return adapter.collectOptical(
		cfg,
		sampledAt,
	)
}

func (adapter fakeONUVendorAdapter) BuildPersistenceCandidates(
	ifmib *snmpmonitor.IFMIBCollection,
	optical *snmpmonitor.ONUOpticalCollection,
) ([]snmpmonitor.ONUPersistenceCandidate, error) {
	if adapter.buildCandidates == nil {
		return nil, errors.New(
			"fake ONU candidate builder is not configured",
		)
	}

	return adapter.buildCandidates(
		ifmib,
		optical,
	)
}

func TestPollNetworkDeviceSNMPv2cOLTCollectsAndPersistsONU(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	in := uint64(1_000_000)
	out := uint64(2_000_000)

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 21},
		DeviceType:         "OLT",
		Vendor:             "VSOL",
		ManagementIP:       "192.0.2.60",
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
		8,
		0,
		0,
		0,
		time.UTC,
	)

	opticalCalls := 0
	onuPersistCalls := 0

	adapter := fakeONUVendorAdapter{
		name: "VSOL",

		collectOptical: func(
			cfg snmpmonitor.V2CConfig,
			gotSampledAt time.Time,
		) (*snmpmonitor.ONUOpticalCollection, error) {
			opticalCalls++

			if cfg.Community != "tscloud" {
				t.Fatal(
					"unexpected ONU collector community",
				)
			}

			if !gotSampledAt.Equal(sampledAt) {
				t.Fatal(
					"unexpected ONU sample time",
				)
			}

			rx := -12.64

			return &snmpmonitor.ONUOpticalCollection{
				Vendor:    "VSOL",
				SampledAt: sampledAt,
				Records: []snmpmonitor.ONUOpticalRecord{
					{
						PONNo:      1,
						ONUNo:      2,
						RxPowerDBM: &rx,
					},
				},
			}, nil
		},

		buildCandidates: func(
			ifmib *snmpmonitor.IFMIBCollection,
			optical *snmpmonitor.ONUOpticalCollection,
		) ([]snmpmonitor.ONUPersistenceCandidate, error) {
			if ifmib == nil {
				t.Fatal(
					"expected IF-MIB collection",
				)
			}

			if optical == nil ||
				len(optical.Records) != 1 {
				t.Fatal(
					"expected optical collection",
				)
			}

			return []snmpmonitor.ONUPersistenceCandidate{
				{
					PONNo:      1,
					ONUNo:      2,
					IfIndex:    14,
					OperStatus: "UP",
					InOctets:   in,
					OutOctets:  out,
					SampledAt:  sampledAt,
				},
			}, nil
		},
	}

	result, err := pollNetworkDeviceSNMPv2c(
		device,
		key,
		sampledAt,
		networkDevicePollDeps{
			probe: func(
				string,
				int,
				string,
			) (string, error) {
				return "ONLINE", nil
			},

			collect: func(
				_ snmpmonitor.V2CConfig,
				gotSampledAt time.Time,
			) (*snmpmonitor.IFMIBCollection, error) {
				return &snmpmonitor.IFMIBCollection{
					SampledAt: gotSampledAt,
					Ports: []snmpmonitor.IFMIBPort{
						{
							IfIndex:     14,
							Name:        "EPON01ONU2",
							OperStatus:  1,
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
				return nil
			},

			getSysObjectID: func(
				cfg snmpmonitor.V2CConfig,
			) (string, error) {
				if cfg.Community != "tscloud" {
					t.Fatal(
						"unexpected sysObjectID community",
					)
				}

				return ".1.3.6.1.4.1.37950.1.1.5.10.14.1",
					nil
			},

			resolveONUAdapter: func(
				vendor string,
				sysObjectID string,
			) (snmpmonitor.ONUVendorAdapter, bool) {
				if vendor != "VSOL" {
					t.Fatalf(
						"vendor=%q want=VSOL",
						vendor,
					)
				}

				if sysObjectID == "" {
					t.Fatal(
						"sysObjectID must not be empty",
					)
				}

				return adapter, true
			},

			persistONU: func(
				networkDeviceID uint,
				candidates []snmpmonitor.ONUPersistenceCandidate,
			) error {
				onuPersistCalls++

				if networkDeviceID != 21 {
					t.Fatalf(
						"device ID=%d want=21",
						networkDeviceID,
					)
				}

				if len(candidates) != 1 {
					t.Fatalf(
						"ONU candidate count=%d want=1",
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
			"status=%q want=ONLINE",
			result.Status,
		)
	}

	if result.TelemetryError != nil {
		t.Fatalf(
			"unexpected generic telemetry error: %v",
			result.TelemetryError,
		)
	}

	if result.ONUError != nil {
		t.Fatalf(
			"unexpected ONU error: %v",
			result.ONUError,
		)
	}

	if result.PortCount != 1 {
		t.Fatalf(
			"port count=%d want=1",
			result.PortCount,
		)
	}

	if result.ONUCount != 1 {
		t.Fatalf(
			"ONU count=%d want=1",
			result.ONUCount,
		)
	}

	if result.ONUAdapter != "VSOL" {
		t.Fatalf(
			"ONU adapter=%q want=VSOL",
			result.ONUAdapter,
		)
	}

	if opticalCalls != 1 ||
		onuPersistCalls != 1 {
		t.Fatalf(
			"unexpected calls optical=%d persistONU=%d",
			opticalCalls,
			onuPersistCalls,
		)
	}
}

func TestPollNetworkDeviceSNMPv2cONUFailureDoesNotBreakPortTelemetry(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	in := uint64(100)
	out := uint64(200)

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 22},
		DeviceType:         "OLT",
		Vendor:             "VSOL",
		ManagementIP:       "192.0.2.61",
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		SNMPSecretEncrypted: encryptedPollTestCommunity(
			t,
			key,
		),
	}

	portPersisted := false

	adapter := fakeONUVendorAdapter{
		name: "VSOL",

		collectOptical: func(
			snmpmonitor.V2CConfig,
			time.Time,
		) (*snmpmonitor.ONUOpticalCollection, error) {
			return nil, errors.New(
				"optical collector unavailable",
			)
		},
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
				_ snmpmonitor.V2CConfig,
				sampledAt time.Time,
			) (*snmpmonitor.IFMIBCollection, error) {
				return &snmpmonitor.IFMIBCollection{
					SampledAt: sampledAt,
					Ports: []snmpmonitor.IFMIBPort{
						{
							IfIndex:     1,
							Name:        "GE0/1",
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
				portPersisted = true
				return nil
			},

			getSysObjectID: func(
				snmpmonitor.V2CConfig,
			) (string, error) {
				return ".1.3.6.1.4.1.37950.1",
					nil
			},

			resolveONUAdapter: func(
				string,
				string,
			) (snmpmonitor.ONUVendorAdapter, bool) {
				return adapter, true
			},

			persistONU: func(
				uint,
				[]snmpmonitor.ONUPersistenceCandidate,
			) error {
				t.Fatal(
					"ONU persistence must not run after optical failure",
				)
				return nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !portPersisted {
		t.Fatal(
			"generic port telemetry was not persisted",
		)
	}

	if result.Status != "ONLINE" {
		t.Fatalf(
			"status=%q want=ONLINE",
			result.Status,
		)
	}

	if result.TelemetryError != nil {
		t.Fatalf(
			"generic telemetry must remain successful: %v",
			result.TelemetryError,
		)
	}

	if result.ONUError == nil {
		t.Fatal(
			"expected isolated ONU error",
		)
	}

	if result.PortCount != 1 {
		t.Fatalf(
			"port count=%d want=1",
			result.PortCount,
		)
	}

	if result.ONUCount != 0 {
		t.Fatalf(
			"ONU count=%d want=0",
			result.ONUCount,
		)
	}

	if result.ONUAdapter != "VSOL" {
		t.Fatalf(
			"ONU adapter=%q want=VSOL",
			result.ONUAdapter,
		)
	}
}

func TestPollNetworkDeviceSNMPv2cUnsupportedONUAdapterIsNonFatal(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	in := uint64(100)
	out := uint64(200)

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 23},
		DeviceType:         "OLT",
		Vendor:             "UNKNOWN",
		ManagementIP:       "192.0.2.62",
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
				return nil
			},

			getSysObjectID: func(
				snmpmonitor.V2CConfig,
			) (string, error) {
				return ".1.3.6.1.4.1.99999.1",
					nil
			},

			resolveONUAdapter: func(
				string,
				string,
			) (snmpmonitor.ONUVendorAdapter, bool) {
				return nil, false
			},

			persistONU: func(
				uint,
				[]snmpmonitor.ONUPersistenceCandidate,
			) error {
				t.Fatal(
					"unsupported adapter must not persist ONU telemetry",
				)
				return nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.TelemetryError != nil ||
		result.ONUError != nil {
		t.Fatalf(
			"unsupported adapter must be non-fatal: telemetry=%v onu=%v",
			result.TelemetryError,
			result.ONUError,
		)
	}

	if result.PortCount != 1 {
		t.Fatalf(
			"port count=%d want=1",
			result.PortCount,
		)
	}

	if result.ONUCount != 0 ||
		result.ONUAdapter != "" {
		t.Fatalf(
			"unexpected ONU result count=%d adapter=%q",
			result.ONUCount,
			result.ONUAdapter,
		)
	}
}

func TestPollNetworkDeviceSNMPv2cNonOLTSkipsONUPath(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	in := uint64(10)
	out := uint64(20)

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 24},
		DeviceType:         "SWITCH",
		Vendor:             "VSOL",
		ManagementIP:       "192.0.2.63",
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
				return nil
			},

			getSysObjectID: func(
				snmpmonitor.V2CConfig,
			) (string, error) {
				t.Fatal(
					"non-OLT must not read sysObjectID for ONU path",
				)
				return "", nil
			},

			resolveONUAdapter: func(
				string,
				string,
			) (snmpmonitor.ONUVendorAdapter, bool) {
				t.Fatal(
					"non-OLT must not resolve ONU adapter",
				)
				return nil, false
			},

			persistONU: func(
				uint,
				[]snmpmonitor.ONUPersistenceCandidate,
			) error {
				t.Fatal(
					"non-OLT must not persist ONU telemetry",
				)
				return nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.PortCount != 1 ||
		result.ONUCount != 0 ||
		result.ONUError != nil {
		t.Fatalf(
			"unexpected non-OLT result ports=%d onus=%d onu_error=%v",
			result.PortCount,
			result.ONUCount,
			result.ONUError,
		)
	}
}

func TestPollNetworkDeviceSNMPv2cOpticalFailureFallsBackToIFMIBONU(
	t *testing.T,
) {
	key := "01234567890123456789012345678901"

	in := uint64(1000)
	out := uint64(2000)

	device := &models.NetworkDevice{
		Model:              gorm.Model{ID: 25},
		DeviceType:         "OLT",
		Vendor:             "VSOL",
		ManagementIP:       "192.0.2.64",
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
		12,
		45,
		0,
		0,
		time.UTC,
	)

	persistCalls := 0

	adapter := fakeONUVendorAdapter{
		name: "VSOL",

		collectOptical: func(
			cfg snmpmonitor.V2CConfig,
			_ time.Time,
		) (*snmpmonitor.ONUOpticalCollection, error) {
			if cfg.Timeout != 10*time.Second {
				t.Fatalf(
					"optical timeout=%s want=10s",
					cfg.Timeout,
				)
			}

			if cfg.Retries != 1 {
				t.Fatalf(
					"optical retries=%d want=1",
					cfg.Retries,
				)
			}

			return nil, errors.New("optical timeout")
		},

		buildCandidates: func(
			ifmib *snmpmonitor.IFMIBCollection,
			optical *snmpmonitor.ONUOpticalCollection,
		) ([]snmpmonitor.ONUPersistenceCandidate, error) {
			if ifmib == nil {
				t.Fatal("expected IF-MIB collection")
			}

			if optical != nil {
				t.Fatal("expected nil optical fallback")
			}

			return []snmpmonitor.ONUPersistenceCandidate{
				{
					PONNo:       1,
					ONUNo:       2,
					IfIndex:     14,
					Description: "EPON01ONU2",
					OperStatus:  "UP",
					InOctets:    in,
					OutOctets:   out,
					SampledAt:   sampledAt,
				},
			}, nil
		},
	}

	result, err := pollNetworkDeviceSNMPv2c(
		device,
		key,
		sampledAt,
		networkDevicePollDeps{
			probe: func(
				string,
				int,
				string,
			) (string, error) {
				return "ONLINE", nil
			},

			collect: func(
				cfg snmpmonitor.V2CConfig,
				gotSampledAt time.Time,
			) (*snmpmonitor.IFMIBCollection, error) {
				if cfg.Timeout != 3*time.Second {
					t.Fatalf(
						"generic timeout=%s want=3s",
						cfg.Timeout,
					)
				}

				if cfg.Retries != 0 {
					t.Fatalf(
						"generic retries=%d want=0",
						cfg.Retries,
					)
				}

				return &snmpmonitor.IFMIBCollection{
					SampledAt: gotSampledAt,
					Ports: []snmpmonitor.IFMIBPort{
						{
							IfIndex:     14,
							Name:        "EPON01ONU2",
							OperStatus:  1,
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
				return nil
			},

			getSysObjectID: func(
				snmpmonitor.V2CConfig,
			) (string, error) {
				return ".1.3.6.1.4.1.37950.1.1.5.10.14.1", nil
			},

			resolveONUAdapter: func(
				string,
				string,
			) (snmpmonitor.ONUVendorAdapter, bool) {
				return adapter, true
			},

			persistONU: func(
				networkDeviceID uint,
				candidates []snmpmonitor.ONUPersistenceCandidate,
			) error {
				persistCalls++

				if networkDeviceID != 25 {
					t.Fatalf(
						"device ID=%d want=25",
						networkDeviceID,
					)
				}

				if len(candidates) != 1 {
					t.Fatalf(
						"candidate count=%d want=1",
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
			"status=%q want=ONLINE",
			result.Status,
		)
	}

	if result.TelemetryError != nil {
		t.Fatalf(
			"unexpected generic telemetry error: %v",
			result.TelemetryError,
		)
	}

	if result.ONUError == nil {
		t.Fatal("expected optical warning in ONUError")
	}

	if result.ONUCount != 1 {
		t.Fatalf(
			"ONU count=%d want=1",
			result.ONUCount,
		)
	}

	if result.ONUAdapter != "VSOL" {
		t.Fatalf(
			"ONU adapter=%q want=VSOL",
			result.ONUAdapter,
		)
	}

	if persistCalls != 1 {
		t.Fatalf(
			"persist calls=%d want=1",
			persistCalls,
		)
	}
}
