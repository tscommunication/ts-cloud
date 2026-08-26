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

	getSysObjectID func(
		cfg snmpmonitor.V2CConfig,
	) (string, error)

	resolveONUAdapter func(
		vendor string,
		sysObjectID string,
	) (snmpmonitor.ONUVendorAdapter, bool)

	persistONU func(
		networkDeviceID uint,
		candidates []snmpmonitor.ONUPersistenceCandidate,
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

		getSysObjectID: getSNMPv2cSysObjectID,

		resolveONUAdapter: snmpmonitor.ResolveONUVendorAdapter,

		persistONU: PersistNetworkDeviceONUCandidates,
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

	if strings.ToUpper(
		strings.TrimSpace(device.DeviceType),
	) != "OLT" {
		return result, nil
	}

	if deps.getSysObjectID == nil ||
		deps.resolveONUAdapter == nil ||
		deps.persistONU == nil {
		result.ONUError = errors.New(
			"ONU poll dependencies are incomplete",
		)
		return result, nil
	}

	cfg := snmpmonitor.V2CConfig{
		Host:      device.ManagementIP,
		Port:      uint16(device.SNMPPort),
		Community: community,
		Timeout:   3 * time.Second,
		Retries:   0,
	}

	sysObjectID, err := deps.getSysObjectID(cfg)
	if err != nil {
		result.ONUError = fmt.Errorf(
			"read sysObjectID for ONU adapter: %w",
			err,
		)
		return result, nil
	}

	adapter, supported := deps.resolveONUAdapter(
		device.Vendor,
		sysObjectID,
	)

	if !supported || adapter == nil {
		return result, nil
	}

	result.ONUAdapter = adapter.Name()

	opticalCfg := cfg
	opticalCfg.Timeout = 10 * time.Second
	opticalCfg.Retries = 1

	optical, err := adapter.CollectOptical(
		opticalCfg,
		sampledAt,
	)
	if err != nil {
		result.ONUError = fmt.Errorf(
			"collect %s ONU optical telemetry: %w",
			result.ONUAdapter,
			err,
		)
		optical = nil
	}

	onuCandidates, err :=
		adapter.BuildPersistenceCandidates(
			collection,
			optical,
		)
	if err != nil {
		result.ONUError = fmt.Errorf(
			"build %s ONU persistence candidates: %w",
			result.ONUAdapter,
			err,
		)
		return result, nil
	}

	if inventoryCollector, ok :=
		adapter.(snmpmonitor.BDCOMONUInventoryCollector); ok {
		inventoryCfg := cfg
		inventoryCfg.Timeout = 8 * time.Second
		inventoryCfg.Retries = 0

		inventory, inventoryErr :=
			inventoryCollector.CollectInventory(
				inventoryCfg,
				sampledAt,
			)

		if inventoryErr == nil {
			onuCandidates =
				snmpmonitor.MergeBDCOMONUInventory(
					onuCandidates,
					inventory,
				)
		}
	}

	if registrationCollector, ok :=
		adapter.(snmpmonitor.ONURegistrationTimeCollector); ok {
		registrationCfg := cfg
		registrationCfg.Timeout = 8 * time.Second
		registrationCfg.Retries = 0

		dhaka := time.FixedZone(
			"Asia/Dhaka",
			6*60*60,
		)

		registrationRecords, registrationErr :=
			registrationCollector.CollectRegistrationTimes(
				registrationCfg,
				dhaka,
			)

		if registrationErr == nil {
			onuCandidates =
				snmpmonitor.MergeVSOLONURegistrationTimes(
					onuCandidates,
					registrationRecords,
				)
		}
	}

	if len(onuCandidates) == 0 {
		result.ONUError = fmt.Errorf(
			"%s ONU persistence candidates are empty",
			result.ONUAdapter,
		)
		return result, nil
	}

	if err := deps.persistONU(
		device.ID,
		onuCandidates,
	); err != nil {
		result.ONUError = fmt.Errorf(
			"persist %s ONU telemetry: %w",
			result.ONUAdapter,
			err,
		)
		return result, nil
	}

	result.ONUCount = len(onuCandidates)

	return result, nil
}

func getSNMPv2cSysObjectID(
	cfg snmpmonitor.V2CConfig,
) (string, error) {
	client, err := snmpmonitor.NewV2CClient(cfg)
	if err != nil {
		return "", err
	}

	result, err := snmpmonitor.GetOne(
		client,
		snmpmonitor.SysObjectIDOID,
	)
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", errors.New(
			"sysObjectID response is nil",
		)
	}

	value, err := snmpmonitor.StringValue(
		result.Value,
	)
	if err != nil {
		return "", fmt.Errorf(
			"parse sysObjectID: %w",
			err,
		)
	}

	value = strings.TrimSpace(value)

	if value == "" {
		return "", errors.New(
			"sysObjectID is empty",
		)
	}

	return value, nil
}
