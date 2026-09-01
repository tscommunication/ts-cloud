package services

import (
	"errors"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

// CustomerNetworkPath joins a customer technical profile to OLT inventory only
// when its ONU MAC or serial number matches. For a mapped active PPPoE user,
// RouterOS caller-ID/MAC is also used automatically. It never guesses optical
// values or writes an inferred match back to the customer profile.
type CustomerNetworkPath struct {
	Profile       *models.CustomerTechnicalProfile
	ONU           *models.NetworkDeviceONU
	LatestOptical *models.NetworkDeviceONUSample
}

func GetCustomerNetworkPath(customerID uint) (*CustomerNetworkPath, error) {
	profile, err := repositories.GetCustomerTechnicalProfile(customerID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if profile == nil {
		profile = &models.CustomerTechnicalProfile{CustomerID: customerID}
	}

	path := &CustomerNetworkPath{Profile: profile}
	identities := []string{profile.ONUMAC}
	if session, sessionErr := repositories.GetActiveNetworkRouterPPPoESessionByCustomerID(customerID); sessionErr == nil {
		identities = append(identities, session.CallerID)
	} else if !errors.Is(sessionErr, gorm.ErrRecordNotFound) {
		return nil, sessionErr
	}
	if account, accountErr := GetCustomerInternetAccount(customerID); accountErr == nil {
		identities = append(identities, account.MACAddress)
	} else if !errors.Is(accountErr, gorm.ErrRecordNotFound) {
		return nil, accountErr
	}
	var macAddress string
	for _, identity := range identities {
		if strings.TrimSpace(identity) != "" {
			macAddress = identity
			break
		}
	}
	onu, err := repositories.FindNetworkDeviceONUByIdentity(macAddress, profile.ONUSerial, profile.ONUSN)
	if err != nil || onu == nil {
		return path, err
	}
	path.ONU = onu
	optical, err := repositories.LatestNetworkDeviceONUOpticalSample(onu.ID)
	if err != nil {
		return nil, err
	}
	path.LatestOptical = optical
	return path, nil
}
