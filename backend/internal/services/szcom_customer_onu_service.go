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

type szcomCLIAcrossPONsResolver func(
	ctx context.Context,
	host string,
	port int,
	username string,
	password string,
	ponONUs map[int][]int,
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
		func(
			ctx context.Context,
			host string,
			port int,
			username string,
			password string,
			ponONUs map[int][]int,
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

			return client.ResolveLearnedMACAcrossPONs(
				ctx,
				ponONUs,
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
	cliAcrossResolver szcomCLIAcrossPONsResolver,
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
		cliAcrossResolver == nil ||
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

	inventory, err := onuList(device.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"list SZCOM ONU inventory: %w",
			err,
		)
	}

	resolveOnPON := func(
		ponNo int,
	) (*SZCOMCustomerONUResolution, error) {
		onuByNumber := make(
			map[int]*models.NetworkDeviceONU,
		)
		onuNos := make([]int, 0)

		for i := range inventory {
			onu := &inventory[i]

			if onu.PONNo != ponNo ||
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
			ponNo,
			onuNos,
			cpeMAC,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve SZCOM exact learned customer MAC on PON %d: %w",
				ponNo,
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

		onu.NetworkDevice = device

		return &SZCOMCustomerONUResolution{
			ONU:        onu,
			LearnedMAC: learnedMAC,
		}, nil
	}

	// Fast path: BRIDGE-MIB resolved the exact customer CPE MAC
	// to a PON, so scan only that PON.
	if learnedPON != nil {
		resolution, err := resolveOnPON(learnedPON.PONNo)
		if err != nil {
			return nil, err
		}
		if resolution == nil {
			return nil, nil
		}

		resolution.LearnedPON = learnedPON
		return resolution, nil
	}

	// FDB-miss fallback:
	// Dynamic BRIDGE-MIB entries can age out while the ONU and its
	// learned customer MAC remain available through the read-only CLI.
	//
	// Use one Telnet session for all candidate PONs. This avoids repeated
	// login/enable cycles and lets the CLI resolver reject ambiguity
	// globally across all PONs before returning an ONU.
	ponONUs := make(map[int][]int)
	onuByLocation := make(
		map[[2]int]*models.NetworkDeviceONU,
	)

	for i := range inventory {
		onu := &inventory[i]

		if onu.PONNo <= 0 ||
			onu.ONUNo <= 0 ||
			onu.ONUNo > 64 {
			continue
		}

		key := [2]int{onu.PONNo, onu.ONUNo}
		onuByLocation[key] = onu
		ponONUs[onu.PONNo] = append(
			ponONUs[onu.PONNo],
			onu.ONUNo,
		)
	}

	if len(ponONUs) == 0 {
		return nil, nil
	}

	learnedMAC, err := cliAcrossResolver(
		ctx,
		device.ManagementIP,
		szcommonitor.DefaultTelnetPort,
		strings.TrimSpace(
			device.ManagementUsername,
		),
		managementPassword,
		ponONUs,
		cpeMAC,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve SZCOM FDB-miss fallback: %w",
			err,
		)
	}

	if learnedMAC == nil {
		return nil, nil
	}

	location := [2]int{
		learnedMAC.PONNo,
		learnedMAC.ONUNo,
	}
	onu := onuByLocation[location]
	if onu == nil {
		return nil, nil
	}

	onu.NetworkDevice = device

	return &SZCOMCustomerONUResolution{
		ONU:        onu,
		LearnedMAC: learnedMAC,
	}, nil
}
