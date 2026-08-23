package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type networkDevicePollDeps struct {
	probe func(
		host string,
		port int,
		community string,
	) (string, error)

	collect func(
		cfg snmpmonitor.V2CConfig,
		sampledAt time.Time,
	) (*snmpmonitor.IFMIBCollection, error)

	persist func(
		networkDeviceID uint,
		candidates []snmpmonitor.PortPersistenceCandidate,
	) error
}

func defaultNetworkDevicePollDeps() networkDevicePollDeps {
	return networkDevicePollDeps{
		probe: probeSNMPv2c,
		collect: func(
			cfg snmpmonitor.V2CConfig,
			sampledAt time.Time,
		) (*snmpmonitor.IFMIBCollection, error) {
			return snmpmonitor.CollectIFMIBV2C(
				cfg,
				sampledAt,
			)
		},
		persist: PersistNetworkDevicePortCandidates,
	}
}

type networkDevicePollResult struct {
	Status         string
	ProbeError     error
	TelemetryError error
	ONUError       error
	PortCount      int
	ONUCount       int
	ONUAdapter     string
}

func pollNetworkDeviceSNMPv2c(
	device *models.NetworkDevice,
	keyMaterial string,
	sampledAt time.Time,
	deps networkDevicePollDeps,
) (*networkDevicePollResult, error) {
	if device == nil {
		return nil, errors.New("network device is required")
	}

	if device.ID == 0 {
		return nil, errors.New("network device ID is required")
	}

	if device.MonitoringProtocol != "SNMP" {
		return nil, errors.New(
			"network device monitoring protocol is not SNMP",
		)
	}

	if device.SNMPVersion != "V2C" {
		return nil, errors.New(
			"network device SNMP version is not V2C",
		)
	}

	if sampledAt.IsZero() {
		return nil, errors.New("sample time is required")
	}

	if deps.probe == nil ||
		deps.collect == nil ||
		deps.persist == nil {
		return nil, errors.New(
			"network device poll dependencies are incomplete",
		)
	}

	community, err := security.DecryptSecret(
		device.SNMPSecretEncrypted,
		keyMaterial,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decrypt SNMP community: %w",
			err,
		)
	}

	if strings.TrimSpace(community) == "" {
		return nil, errors.New("SNMP community is empty")
	}

	status, probeErr := deps.probe(
		device.ManagementIP,
		device.SNMPPort,
		community,
	)

	result := &networkDevicePollResult{
		Status:     status,
		ProbeError: probeErr,
	}

	collection, collectErr := deps.collect(
		snmpmonitor.V2CConfig{
			Host:      device.ManagementIP,
			Port:      uint16(device.SNMPPort),
			Community: community,
			Timeout:   3 * time.Second,
			Retries:   0,
		},
		sampledAt,
	)
	if collectErr != nil {
		result.TelemetryError = fmt.Errorf(
			"collect IF-MIB telemetry: %w",
			collectErr,
		)

		return result, nil
	}

	candidates, err :=
		snmpmonitor.BuildPortPersistenceCandidates(
			collection,
		)
	if err != nil {
		result.TelemetryError = fmt.Errorf(
			"build IF-MIB persistence candidates: %w",
			err,
		)

		return result, nil
	}

	if err := deps.persist(
		device.ID,
		candidates,
	); err != nil {
		result.TelemetryError = fmt.Errorf(
			"persist IF-MIB telemetry: %w",
			err,
		)

		return result, nil
	}

	result.PortCount = len(candidates)

	return result, nil
}
