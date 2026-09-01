package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/models"
	hsgqmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/hsgq"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type HSGQCustomerONUResolution struct {
	ONU        *models.NetworkDeviceONU
	LearnedMAC *hsgqmonitor.LearnedMACResolution
}

type hsgqLearnedMACResolver func(
	ctx context.Context,
	device *models.NetworkDevice,
	password string,
	macAddress string,
) (*hsgqmonitor.LearnedMACResolution, error)

type hsgqONUByPositionFinder func(
	deviceID uint,
	ponNo int,
	onuNo int,
) (*models.NetworkDeviceONU, error)

func ResolveHSGQCustomerONU(
	ctx context.Context,
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
) (*HSGQCustomerONUResolution, error) {
	return resolveHSGQCustomerONU(
		ctx,
		device,
		cpeMAC,
		credentialKey,
		resolveHSGQHTTPMAC,
		repositories.FindNetworkDeviceONUByPosition,
	)
}

func resolveHSGQHTTPMAC(
	ctx context.Context,
	device *models.NetworkDevice,
	password string,
	macAddress string,
) (*hsgqmonitor.LearnedMACResolution, error) {
	client, err := hsgqmonitor.NewClient(
		device.ManagementIP,
		device.ManagementPort,
		device.ManagementUsername,
		password,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create HSGQ management client: %w",
			err,
		)
	}

	token, err := client.Login(ctx)
	if err != nil {
		return nil, err
	}

	return client.ResolveLearnedMAC(
		ctx,
		token,
		macAddress,
	)
}

func resolveHSGQCustomerONU(
	ctx context.Context,
	device *models.NetworkDevice,
	cpeMAC string,
	credentialKey string,
	learnedResolver hsgqLearnedMACResolver,
	onuFinder hsgqONUByPositionFinder,
) (*HSGQCustomerONUResolution, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
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
	if strings.ToUpper(
		strings.TrimSpace(device.Vendor),
	) != "HSGQ" {
		return nil, errors.New(
			"network device vendor must be HSGQ",
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
			"HSGQ management username is required",
		)
	}
	if strings.TrimSpace(
		device.ManagementSecretEncrypted,
	) == "" {
		return nil, errors.New(
			"HSGQ management credential is required",
		)
	}
	if device.ManagementPort < 1 ||
		device.ManagementPort > 65535 {
		return nil, errors.New(
			"HSGQ management port is invalid",
		)
	}
	if learnedResolver == nil ||
		onuFinder == nil {
		return nil, errors.New(
			"HSGQ customer ONU resolver dependency is required",
		)
	}

	managementPassword, err :=
		security.DecryptSecret(
			device.ManagementSecretEncrypted,
			credentialKey,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"decrypt HSGQ management credential: %w",
			err,
		)
	}

	learned, err := learnedResolver(
		ctx,
		device,
		managementPassword,
		cpeMAC,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve HSGQ learned customer MAC: %w",
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
			"find correlated HSGQ ONU inventory: %w",
			err,
		)
	}

	if onu == nil {
		return nil, nil
	}

	return &HSGQCustomerONUResolution{
		ONU:        onu,
		LearnedMAC: learned,
	}, nil
}
