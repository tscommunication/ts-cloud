package dto

import "github.com/tscommunication/ts-cloud/internal/models"

type CustomerTechnicalProfileRequest struct {
	ONUMAC      string `json:"onu_mac"`
	OLTPON      string `json:"olt_pon"`
	OLTSlot     string `json:"olt_slot"`
	OLTPort     string `json:"olt_port"`
	ONUType     string `json:"onu_type"`
	ONUModel    string `json:"onu_model"`
	ONUIP       string `json:"onu_ip"`
	ONUPassword string `json:"onu_password"`
	ONUSerial   string `json:"onu_serial"`
	ONUSN       string `json:"onu_sn"`

	RouterBrand    string `json:"router_brand"`
	RouterModel    string `json:"router_model"`
	RouterIP       string `json:"router_ip"`
	MikroTikPort   string `json:"mikrotik_port"`
	RouterPassword string `json:"router_password"`

	CableType   string  `json:"cable_type"`
	CableLength float64 `json:"cable_length"`

	MediaConverterMAC      string `json:"media_converter_mac"`
	MediaConverterIP       string `json:"media_converter_ip"`
	MediaConverterPassword string `json:"media_converter_password"`

	SwitchModel    string `json:"switch_model"`
	SwitchPort     string `json:"switch_port"`
	SwitchIP       string `json:"switch_ip"`
	SwitchPassword string `json:"switch_password"`

	AdditionalNote string `json:"additional_note"`
}

type CustomerTechnicalProfileResponse struct {
	ID         uint `json:"id"`
	CustomerID uint `json:"customer_id"`

	ONUMAC    string `json:"onu_mac"`
	OLTPON    string `json:"olt_pon"`
	OLTSlot   string `json:"olt_slot"`
	OLTPort   string `json:"olt_port"`
	ONUType   string `json:"onu_type"`
	ONUModel  string `json:"onu_model"`
	ONUIP     string `json:"onu_ip"`
	ONUSerial string `json:"onu_serial"`
	ONUSN     string `json:"onu_sn"`

	ONUPasswordConfigured bool `json:"onu_password_configured"`

	RouterBrand              string `json:"router_brand"`
	RouterModel              string `json:"router_model"`
	RouterIP                 string `json:"router_ip"`
	MikroTikPort             string `json:"mikrotik_port"`
	RouterPasswordConfigured bool   `json:"router_password_configured"`

	CableType   string  `json:"cable_type"`
	CableLength float64 `json:"cable_length"`

	MediaConverterMAC                string `json:"media_converter_mac"`
	MediaConverterIP                 string `json:"media_converter_ip"`
	MediaConverterPasswordConfigured bool   `json:"media_converter_password_configured"`

	SwitchModel              string `json:"switch_model"`
	SwitchPort               string `json:"switch_port"`
	SwitchIP                 string `json:"switch_ip"`
	SwitchPasswordConfigured bool   `json:"switch_password_configured"`

	AdditionalNote string `json:"additional_note"`
}

func ToCustomerTechnicalProfileResponse(
	profile models.CustomerTechnicalProfile,
) CustomerTechnicalProfileResponse {
	return CustomerTechnicalProfileResponse{
		ID:         profile.ID,
		CustomerID: profile.CustomerID,

		ONUMAC:    profile.ONUMAC,
		OLTPON:    profile.OLTPON,
		OLTSlot:   profile.OLTSlot,
		OLTPort:   profile.OLTPort,
		ONUType:   profile.ONUType,
		ONUModel:  profile.ONUModel,
		ONUIP:     profile.ONUIP,
		ONUSerial: profile.ONUSerial,
		ONUSN:     profile.ONUSN,

		ONUPasswordConfigured: profile.ONUPasswordEncrypted != "",

		RouterBrand:              profile.RouterBrand,
		RouterModel:              profile.RouterModel,
		RouterIP:                 profile.RouterIP,
		MikroTikPort:             profile.MikroTikPort,
		RouterPasswordConfigured: profile.RouterPasswordEncrypted != "",

		CableType:   profile.CableType,
		CableLength: profile.CableLength,

		MediaConverterMAC:                profile.MediaConverterMAC,
		MediaConverterIP:                 profile.MediaConverterIP,
		MediaConverterPasswordConfigured: profile.MediaConverterPasswordEncrypted != "",

		SwitchModel:              profile.SwitchModel,
		SwitchPort:               profile.SwitchPort,
		SwitchIP:                 profile.SwitchIP,
		SwitchPasswordConfigured: profile.SwitchPasswordEncrypted != "",

		AdditionalNote: profile.AdditionalNote,
	}
}
