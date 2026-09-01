package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	ecommonitor "github.com/tscommunication/ts-cloud/internal/monitoring/ecom"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type ECOMCustomerONUResolution struct {
	ONU        *models.NetworkDeviceONU
	LearnedMAC *snmpmonitor.ECOMLearnedMACResolution
	HTTP       *ecommonitor.ExactONUResolution
}

type ecomLearnedMACResolver func(
	cfg snmpmonitor.V2CConfig,
	macAddress string,
) (*snmpmonitor.ECOMLearnedMACResolution, error)

type ecomExactONUResolver func(
	ctx context.Context,
	device *models.NetworkDevice,
	password string,
	macAddress string,
	evidence ecommonitor.LearnedMACEvidence,
) (*ecommonitor.ExactONUResolution, error)

type ecomONUByPositionFinder func(
	deviceID uint,
	ponNo int,
	onuNo int,
) (*models.NetworkDeviceONU, error)

func ResolveECOMCustomerONU(
	ctx context.Context,
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
) (*ECOMCustomerONUResolution, error) {
	return resolveECOMCustomerONU(
		ctx,
		device,
		cpeMAC,
		credentialKey,
		func(
			cfg snmpmonitor.V2CConfig,
			macAddress string,
		) (*snmpmonitor.ECOMLearnedMACResolution, error) {
			return (snmpmonitor.ECOMONUAdapter{}).
				ResolveLearnedMAC(cfg, macAddress)
		},
		resolveECOMHTTPONU,
		repositories.FindNetworkDeviceONUByPosition,
	)
}

func resolveECOMHTTPONU(
	ctx context.Context,
	device *models.NetworkDevice,
	password string,
	macAddress string,
	evidence ecommonitor.LearnedMACEvidence,
) (*ecommonitor.ExactONUResolution, error) {
	client, err := ecommonitor.NewClient(
		device.ManagementIP,
		device.ManagementPort,
		device.ManagementUsername,
		password,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create ECOM management client: %w",
			err,
		)
	}

	token, err := client.Login(ctx)
	if err != nil {
		return nil, err
	}

	return client.FindONUByMACWithEvidence(
		ctx,
		token,
		macAddress,
		&evidence,
	)
}

func resolveECOMCustomerONU(
	ctx context.Context,
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
	learnedResolver ecomLearnedMACResolver,
	httpResolver ecomExactONUResolver,
	onuFinder ecomONUByPositionFinder,
) (*ECOMCustomerONUResolution, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if device == nil || device.ID == 0 {
		return nil, errors.New("network device is required")
	}
	if strings.ToUpper(strings.TrimSpace(device.DeviceType)) != "OLT" {
		return nil, errors.New("network device must be an OLT")
	}
	if strings.ToUpper(strings.TrimSpace(device.Vendor)) != "ECOM" {
		return nil, errors.New("network device vendor must be ECOM")
	}
	if strings.ToUpper(strings.TrimSpace(device.MonitoringProtocol)) != "SNMP" {
		return nil, errors.New(
			"ECOM customer ONU resolution requires SNMP monitoring",
		)
	}
	if strings.ToUpper(strings.TrimSpace(device.SNMPVersion)) != "V2C" {
		return nil, errors.New(
			"ECOM customer ONU resolution currently requires SNMP V2C",
		)
	}
	if strings.TrimSpace(cpeMAC) == "" {
		return nil, errors.New("customer CPE MAC is required")
	}
	if strings.TrimSpace(credentialKey) == "" {
		return nil, errors.New("credential key is required")
	}
	if learnedResolver == nil || httpResolver == nil || onuFinder == nil {
		return nil, errors.New(
			"ECOM customer ONU resolver dependency is required",
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

	managementPassword, err := security.DecryptSecret(
		device.ManagementSecretEncrypted,
		credentialKey,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decrypt management credential: %w",
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
			"resolve ECOM learned customer MAC: %w",
			err,
		)
	}
	if learned == nil {
		return nil, nil
	}

	httpResolution, err := httpResolver(
		ctx,
		device,
		managementPassword,
		cpeMAC,
		ecommonitor.LearnedMACEvidence{
			PortID:    learned.PortID,
			Interface: learned.Interface,
			PONNo:     learned.PONNo,
			VLAN:      learned.VLAN,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve ECOM exact ONU: %w",
			err,
		)
	}
	if httpResolution == nil {
		return nil, nil
	}

	if httpResolution.PONNo != learned.PONNo {
		return nil, fmt.Errorf(
			"ECOM exact ONU PON %d does not match learned PON %d",
			httpResolution.PONNo,
			learned.PONNo,
		)
	}

	onu, err := onuFinder(
		device.ID,
		httpResolution.PONNo,
		httpResolution.ONUNo,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find correlated ONU inventory: %w",
			err,
		)
	}
	if onu == nil {
		return nil, nil
	}

	return &ECOMCustomerONUResolution{
		ONU:        onu,
		LearnedMAC: learned,
		HTTP:       httpResolution,
	}, nil
}
