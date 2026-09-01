package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type VSOLCustomerONUResolution struct {
	ONU        *models.NetworkDeviceONU
	LearnedMAC *snmpmonitor.VSOLLearnedMACResolution
}

type vsolLearnedMACResolver func(
	cfg snmpmonitor.V2CConfig,
	macAddress string,
) (*snmpmonitor.VSOLLearnedMACResolution, error)

type vsolONUByPositionFinder func(
	deviceID uint,
	ponNo int,
	onuNo int,
) (*models.NetworkDeviceONU, error)

func ResolveVSOLCustomerONU(
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
) (*VSOLCustomerONUResolution, error) {
	return resolveVSOLCustomerONU(
		device,
		cpeMAC,
		credentialKey,
		func(
			cfg snmpmonitor.V2CConfig,
			macAddress string,
		) (*snmpmonitor.VSOLLearnedMACResolution, error) {
			return (snmpmonitor.VSOLONUAdapter{}).
				ResolveLearnedMAC(cfg, macAddress)
		},
		repositories.FindNetworkDeviceONUByPosition,
	)
}

func resolveVSOLCustomerONU(
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
	learnedResolver vsolLearnedMACResolver,
	onuFinder vsolONUByPositionFinder,
) (*VSOLCustomerONUResolution, error) {
	if device == nil || device.ID == 0 {
		return nil, errors.New("network device is required")
	}

	if strings.ToUpper(strings.TrimSpace(device.DeviceType)) != "OLT" {
		return nil, errors.New("network device must be an OLT")
	}

	if !(snmpmonitor.VSOLONUAdapter{}).Matches(
		device.Vendor,
		"",
	) {
		return nil, errors.New(
			"network device vendor is not VSOL-compatible",
		)
	}

	if strings.ToUpper(
		strings.TrimSpace(device.MonitoringProtocol),
	) != "SNMP" {
		return nil, errors.New(
			"VSOL customer ONU resolution requires SNMP monitoring",
		)
	}

	if strings.ToUpper(
		strings.TrimSpace(device.SNMPVersion),
	) != "V2C" {
		return nil, errors.New(
			"VSOL customer ONU resolution currently requires SNMP V2C",
		)
	}

	if strings.TrimSpace(cpeMAC) == "" {
		return nil, errors.New("customer CPE MAC is required")
	}

	if strings.TrimSpace(credentialKey) == "" {
		return nil, errors.New("credential key is required")
	}

	if learnedResolver == nil || onuFinder == nil {
		return nil, errors.New(
			"VSOL customer ONU resolver dependency is required",
		)
	}

	community, err := security.DecryptSecret(
		device.SNMPSecretEncrypted,
		credentialKey,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decrypt SNMP community: %w",
			err,
		)
	}

	learned, err := learnedResolver(
		snmpmonitor.V2CConfig{
			Host:      device.ManagementIP,
			Port:      uint16(device.SNMPPort),
			Community: community,
			Timeout:   3 * time.Second,
			Retries:   0,
		},
		cpeMAC,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve VSOL learned customer MAC: %w",
			err,
		)
	}

	if learned == nil {
		return nil, nil
	}

	onu, err := onuFinder(
		device.ID,
		learned.PONNo,
		learned.ONUNo,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find correlated VSOL ONU inventory: %w",
			err,
		)
	}

	if onu == nil {
		return nil, nil
	}

	return &VSOLCustomerONUResolution{
		ONU:        onu,
		LearnedMAC: learned,
	}, nil
}
