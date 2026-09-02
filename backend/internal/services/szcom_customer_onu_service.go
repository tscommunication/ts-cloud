package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	szcommonitor "github.com/tscommunication/ts-cloud/internal/monitoring/szcom"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type SZCOMCustomerONUResolution struct {
	ONU        *models.NetworkDeviceONU
	LearnedPON *snmpmonitor.SZCOMLearnedMACPONResolution
	LearnedMAC *szcommonitor.LearnedMACResolution
}

type szcomPONResolver func(
	cfg snmpmonitor.V2CConfig,
	macAddress string,
) (*snmpmonitor.SZCOMLearnedMACPONResolution, error)

type szcomCLIResolver func(
	ctx context.Context,
	host string,
	port int,
	username string,
	password string,
	ponNo int,
	onuNos []int,
	macAddress string,
) (*szcommonitor.LearnedMACResolution, error)

type szcomONUListFunc func(
	deviceID uint,
) ([]models.NetworkDeviceONU, error)

func ResolveSZCOMCustomerONU(
	ctx context.Context,
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
) (*SZCOMCustomerONUResolution, error) {
	return resolveSZCOMCustomerONU(
		ctx,
		device,
		cpeMAC,
		credentialKey,
		func(
			cfg snmpmonitor.V2CConfig,
			macAddress string,
		) (*snmpmonitor.SZCOMLearnedMACPONResolution, error) {
			return (snmpmonitor.SZCOMONUAdapter{}).
				ResolveLearnedMACPON(cfg, macAddress)
		},
		func(
			ctx context.Context,
			host string,
			port int,
			username string,
			password string,
			ponNo int,
			onuNos []int,
			macAddress string,
		) (*szcommonitor.LearnedMACResolution, error) {
			client, err := szcommonitor.NewClient(
				host,
				port,
				username,
				password,
			)
			if err != nil {
				return nil, err
			}

			return client.ResolveLearnedMAC(
				ctx,
				ponNo,
				onuNos,
				macAddress,
			)
		},
		repositories.ListNetworkDeviceONUs,
	)
}

func resolveSZCOMCustomerONU(
	ctx context.Context,
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
	ponResolver szcomPONResolver,
	cliResolver szcomCLIResolver,
	onuList szcomONUListFunc,
) (*SZCOMCustomerONUResolution, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if device == nil || device.ID == 0 {
		return nil, errors.New(
			"network device is required",
		)
	}

	if strings.ToUpper(
		strings.TrimSpace(device.DeviceType),
	) != "OLT" {
		return nil, errors.New(
			"network device must be an OLT",
		)
	}

	if !(snmpmonitor.SZCOMONUAdapter{}).Matches(
		device.Vendor,
		"",
	) {
		return nil, errors.New(
			"network device vendor is not SZCOM-compatible",
		)
	}

	if strings.ToUpper(
		strings.TrimSpace(device.MonitoringProtocol),
	) != "SNMP" ||
		strings.ToUpper(
			strings.TrimSpace(device.SNMPVersion),
		) != "V2C" {
		return nil, errors.New(
			"SZCOM customer ONU resolution requires SNMP V2C",
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

	if strings.TrimSpace(
		device.ManagementUsername,
	) == "" {
		return nil, errors.New(
			"SZCOM management username is required",
		)
	}

	if strings.TrimSpace(
		device.ManagementSecretEncrypted,
	) == "" {
		return nil, errors.New(
			"SZCOM management credential is required",
		)
	}

	if ponResolver == nil ||
		cliResolver == nil ||
		onuList == nil {
		return nil, errors.New(
			"SZCOM customer ONU resolver dependency is required",
		)
	}

	community, err := security.DecryptSecret(
		device.SNMPSecretEncrypted,
		credentialKey,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decrypt SZCOM SNMP community: %w",
			err,
		)
	}

	managementPassword, err := security.DecryptSecret(
		device.ManagementSecretEncrypted,
		credentialKey,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decrypt SZCOM management credential: %w",
			err,
		)
	}

	learnedPON, err := ponResolver(
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
			"resolve SZCOM learned customer MAC PON: %w",
			err,
		)
	}

	if learnedPON == nil {
		return nil, nil
	}

	inventory, err := onuList(device.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"list SZCOM ONU inventory: %w",
			err,
		)
	}

	onuByNumber := make(
		map[int]*models.NetworkDeviceONU,
	)
	onuNos := make([]int, 0)

	for i := range inventory {
		onu := &inventory[i]

		if onu.PONNo != learnedPON.PONNo ||
			onu.ONUNo <= 0 ||
			onu.ONUNo > 64 {
			continue
		}

		onuByNumber[onu.ONUNo] = onu
		onuNos = append(onuNos, onu.ONUNo)
	}

	if len(onuNos) == 0 {
		return nil, nil
	}

	sort.Ints(onuNos)

	learnedMAC, err := cliResolver(
		ctx,
		device.ManagementIP,
		szcommonitor.DefaultTelnetPort,
		strings.TrimSpace(
			device.ManagementUsername,
		),
		managementPassword,
		learnedPON.PONNo,
		onuNos,
		cpeMAC,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve SZCOM exact learned customer MAC: %w",
			err,
		)
	}

	if learnedMAC == nil {
		return nil, nil
	}

	onu := onuByNumber[learnedMAC.ONUNo]
	if onu == nil {
		return nil, nil
	}

	return &SZCOMCustomerONUResolution{
		ONU:        onu,
		LearnedPON: learnedPON,
		LearnedMAC: learnedMAC,
	}, nil
}
