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

type BDCOMCustomerONUResolution struct {
	ONU        *models.NetworkDeviceONU
	LearnedMAC *snmpmonitor.BDCOMLearnedMACResolution
}

type bdcomLearnedMACResolver func(
	cfg snmpmonitor.V2CConfig,
	macAddress string,
) (*snmpmonitor.BDCOMLearnedMACResolution, error)

type bdcomONUByPositionFinder func(
	deviceID uint,
	ponNo int,
	onuNo int,
) (*models.NetworkDeviceONU, error)

func ResolveBDCOMCustomerONU(
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
) (*BDCOMCustomerONUResolution, error) {
	return resolveBDCOMCustomerONU(
		device,
		cpeMAC,
		credentialKey,
		func(
			cfg snmpmonitor.V2CConfig,
			macAddress string,
		) (*snmpmonitor.BDCOMLearnedMACResolution, error) {
			return (snmpmonitor.BDCOMONUAdapter{}).
				ResolveLearnedMAC(cfg, macAddress)
		},
		repositories.FindNetworkDeviceONUByPosition,
	)
}

func resolveBDCOMCustomerONU(
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
	learnedResolver bdcomLearnedMACResolver,
	onuFinder bdcomONUByPositionFinder,
) (*BDCOMCustomerONUResolution, error) {
	if device == nil || device.ID == 0 {
		return nil, errors.New("network device is required")
	}

	if strings.ToUpper(
		strings.TrimSpace(device.DeviceType),
	) != "OLT" {
		return nil, errors.New(
			"network device must be an OLT",
		)
	}

	if !(snmpmonitor.BDCOMONUAdapter{}).Matches(
		device.Vendor,
		"",
	) {
		return nil, errors.New(
			"network device vendor is not BDCOM",
		)
	}

	if strings.ToUpper(
		strings.TrimSpace(device.MonitoringProtocol),
	) != "SNMP" {
		return nil, errors.New(
			"BDCOM customer ONU resolution requires SNMP monitoring",
		)
	}

	if strings.ToUpper(
		strings.TrimSpace(device.SNMPVersion),
	) != "V2C" {
		return nil, errors.New(
			"BDCOM customer ONU resolution currently requires SNMP V2C",
		)
	}

	if strings.TrimSpace(cpeMAC) == "" {
		return nil, errors.New(
			"customer CPE MAC is required",
		)
	}

	if strings.TrimSpace(credentialKey) == "" {
		return nil, errors.New(
			"credential key is required",
		)
	}

	if learnedResolver == nil || onuFinder == nil {
		return nil, errors.New(
			"BDCOM customer ONU resolver dependency is required",
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
			"resolve BDCOM learned customer MAC: %w",
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
			"find correlated BDCOM ONU inventory: %w",
			err,
		)
	}

	if onu == nil {
		return nil, nil
	}

	return &BDCOMCustomerONUResolution{
		ONU:        onu,
		LearnedMAC: learned,
	}, nil
}
