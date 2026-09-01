package services

import (
	"context"
	"errors"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

// CustomerNetworkPath joins a customer technical profile to monitored OLT
// inventory. A technician-recorded ONU MAC/serial is treated as a direct ONU
// identity. PPPoE caller-ID/account MAC is treated as a customer CPE MAC and
// may be correlated through vendor-specific learned-MAC/FDB resolution to the
// exact PON/ONU position. No inferred match is written back to the customer profile.
type CustomerNetworkPath struct {
	Profile       *models.CustomerTechnicalProfile
	ONU           *models.NetworkDeviceONU
	LatestOptical *models.NetworkDeviceONUSample
}

func GetCustomerNetworkPath(
	ctx context.Context,
	customerID uint,
	credentialKey string,
) (*CustomerNetworkPath, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	profile, err := repositories.GetCustomerTechnicalProfile(customerID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if profile == nil {
		profile = &models.CustomerTechnicalProfile{
			CustomerID: customerID,
		}
	}

	path := &CustomerNetworkPath{
		Profile: profile,
	}

	// Highest priority: technician-recorded ONU identity.
	// Never treat a PPPoE caller-ID/CPE MAC as an ONU MAC.
	onu, err := repositories.FindNetworkDeviceONUByIdentity(
		profile.ONUMAC,
		profile.ONUSerial,
		profile.ONUSN,
	)
	if err != nil {
		return nil, err
	}
	if onu != nil {
		return populateCustomerNetworkPathONU(path, onu)
	}

	// Resolve the customer CPE/router MAC independently from ONU identity.
	var cpeMAC string

	if session, sessionErr :=
		repositories.GetActiveNetworkRouterPPPoESessionByCustomerID(
			customerID,
		); sessionErr == nil {
		cpeMAC = strings.TrimSpace(session.CallerID)
	} else if !errors.Is(sessionErr, gorm.ErrRecordNotFound) {
		return nil, sessionErr
	}

	if cpeMAC == "" {
		if account, accountErr :=
			GetCustomerInternetAccount(customerID); accountErr == nil {
			cpeMAC = strings.TrimSpace(account.MACAddress)
		} else if !errors.Is(accountErr, gorm.ErrRecordNotFound) {
			return nil, accountErr
		}
	}

	if cpeMAC == "" || strings.TrimSpace(credentialKey) == "" {
		return path, nil
	}

	devices, err := ListNetworkDevices()
	if err != nil {
		return nil, err
	}

	customer, err := repositories.GetCustomerByID(customerID)
	if err != nil {
		return nil, err
	}

	// VSOL/ZIBBIX correlation:
	// Customer POP -> compatible OLT -> learned CPE MAC/FDB ->
	// exact PON/ONU -> local monitored ONU inventory.
	//
	// The POP constraint is important where multiple OLTs exist across
	// the network. A failure or no-match on one compatible OLT is not
	// allowed to fail the customer page.
	if customer.PopID != nil {
		for i := range devices {
			device := &devices[i]

			if strings.ToUpper(
				strings.TrimSpace(device.DeviceType),
			) != "OLT" ||
				device.POPID == nil ||
				*device.POPID != *customer.PopID ||
				strings.ToUpper(
					strings.TrimSpace(device.MonitoringProtocol),
				) != "SNMP" ||
				strings.ToUpper(
					strings.TrimSpace(device.SNMPVersion),
				) != "V2C" ||
				!device.MonitoringEnabled {
				continue
			}

			if !isVSOLCompatibleNetworkDevice(device) {
				continue
			}

			resolution, resolveErr :=
				ResolveVSOLCustomerONU(
					device,
					cpeMAC,
					credentialKey,
				)

			if resolveErr != nil ||
				resolution == nil ||
				resolution.ONU == nil {
				continue
			}

			return populateCustomerNetworkPathONU(
				path,
				resolution.ONU,
			)
		}
	}

	// ECOM correlation:
	// CPE MAC -> SNMP learned MAC/FDB -> PON -> ECOM HTTP API ->
	// exact ONU number -> local monitored ONU inventory.
	//
	// One unreachable or non-matching ECOM OLT must not make the customer page
	// fail. Continue trying the remaining eligible ECOM OLTs.
	for i := range devices {
		device := &devices[i]

		if strings.ToUpper(strings.TrimSpace(device.DeviceType)) != "OLT" ||
			strings.ToUpper(strings.TrimSpace(device.Vendor)) != "ECOM" ||
			strings.ToUpper(strings.TrimSpace(device.MonitoringProtocol)) != "SNMP" ||
			strings.ToUpper(strings.TrimSpace(device.SNMPVersion)) != "V2C" ||
			!device.MonitoringEnabled ||
			strings.TrimSpace(device.ManagementUsername) == "" ||
			strings.TrimSpace(device.ManagementSecretEncrypted) == "" {
			continue
		}

		resolution, resolveErr := ResolveECOMCustomerONU(
			ctx,
			device,
			cpeMAC,
			credentialKey,
		)
		if resolveErr != nil || resolution == nil || resolution.ONU == nil {
			continue
		}

		return populateCustomerNetworkPathONU(
			path,
			resolution.ONU,
		)
	}

	return path, nil
}

func isVSOLCompatibleNetworkDevice(
	device *models.NetworkDevice,
) bool {
	if device == nil {
		return false
	}

	vendor := strings.ToUpper(
		strings.TrimSpace(device.Vendor),
	)

	return vendor == "VSOL" || vendor == "ZIBBIX"
}

func populateCustomerNetworkPathONU(
	path *CustomerNetworkPath,
	onu *models.NetworkDeviceONU,
) (*CustomerNetworkPath, error) {
	path.ONU = onu

	optical, err :=
		repositories.LatestNetworkDeviceONUOpticalSample(onu.ID)
	if err != nil {
		return nil, err
	}

	path.LatestOptical = optical

	return path, nil
}
