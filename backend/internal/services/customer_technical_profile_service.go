package services

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type CustomerTechnicalProfileInput struct {
	ONUMAC      string
	OLTPON      string
	OLTSlot     string
	OLTPort     string
	ONUType     string
	ONUModel    string
	ONUIP       string
	ONUPassword string
	ONUSerial   string
	ONUSN       string

	RouterBrand    string
	RouterModel    string
	RouterIP       string
	MikroTikPort   string
	RouterPassword string

	CableType   string
	CableLength float64

	MediaConverterMAC      string
	MediaConverterIP       string
	MediaConverterPassword string

	SwitchModel    string
	SwitchPort     string
	SwitchIP       string
	SwitchPassword string

	AdditionalNote string
}

func GetCustomerTechnicalProfile(
	customerID uint,
) (*models.CustomerTechnicalProfile, error) {
	return repositories.GetCustomerTechnicalProfile(customerID)
}

func SaveCustomerTechnicalProfile(
	customerID uint,
	input CustomerTechnicalProfileInput,
	keyMaterial string,
) (*models.CustomerTechnicalProfile, error) {
	profile, err := repositories.GetCustomerTechnicalProfile(customerID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		profile = &models.CustomerTechnicalProfile{
			CustomerID: customerID,
		}
	}

	profile.ONUMAC = strings.TrimSpace(input.ONUMAC)
	profile.OLTPON = strings.TrimSpace(input.OLTPON)
	profile.OLTSlot = strings.TrimSpace(input.OLTSlot)
	profile.OLTPort = strings.TrimSpace(input.OLTPort)
	profile.ONUType = strings.TrimSpace(input.ONUType)
	profile.ONUModel = strings.TrimSpace(input.ONUModel)
	profile.ONUIP = strings.TrimSpace(input.ONUIP)
	profile.ONUSerial = strings.TrimSpace(input.ONUSerial)
	profile.ONUSN = strings.TrimSpace(input.ONUSN)

	profile.RouterBrand = strings.TrimSpace(input.RouterBrand)
	profile.RouterModel = strings.TrimSpace(input.RouterModel)
	profile.RouterIP = strings.TrimSpace(input.RouterIP)
	profile.MikroTikPort = strings.TrimSpace(input.MikroTikPort)

	profile.CableType = strings.TrimSpace(input.CableType)
	profile.CableLength = input.CableLength

	profile.MediaConverterMAC = strings.TrimSpace(input.MediaConverterMAC)
	profile.MediaConverterIP = strings.TrimSpace(input.MediaConverterIP)

	profile.SwitchModel = strings.TrimSpace(input.SwitchModel)
	profile.SwitchPort = strings.TrimSpace(input.SwitchPort)
	profile.SwitchIP = strings.TrimSpace(input.SwitchIP)

	profile.AdditionalNote = strings.TrimSpace(input.AdditionalNote)

	if strings.TrimSpace(input.ONUPassword) != "" {
		encrypted, err := security.EncryptSecret(
			input.ONUPassword,
			keyMaterial,
		)
		if err != nil {
			return nil, err
		}
		profile.ONUPasswordEncrypted = encrypted
	}

	if strings.TrimSpace(input.RouterPassword) != "" {
		encrypted, err := security.EncryptSecret(
			input.RouterPassword,
			keyMaterial,
		)
		if err != nil {
			return nil, err
		}
		profile.RouterPasswordEncrypted = encrypted
	}

	if strings.TrimSpace(input.MediaConverterPassword) != "" {
		encrypted, err := security.EncryptSecret(
			input.MediaConverterPassword,
			keyMaterial,
		)
		if err != nil {
			return nil, err
		}
		profile.MediaConverterPasswordEncrypted = encrypted
	}

	if strings.TrimSpace(input.SwitchPassword) != "" {
		encrypted, err := security.EncryptSecret(
			input.SwitchPassword,
			keyMaterial,
		)
		if err != nil {
			return nil, err
		}
		profile.SwitchPasswordEncrypted = encrypted
	}

	if err := repositories.SaveCustomerTechnicalProfile(profile); err != nil {
		return nil, err
	}

	return profile, nil
}
