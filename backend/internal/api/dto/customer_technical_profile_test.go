package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestCustomerTechnicalProfileResponseDoesNotExposeSecrets(t *testing.T) {
	response := ToCustomerTechnicalProfileResponse(
		models.CustomerTechnicalProfile{
			ID:                              1,
			CustomerID:                      10,
			MikroTikPort:                    "sfp-sfpplus1",
			ONUPasswordEncrypted:            "encrypted-onu-secret",
			RouterPasswordEncrypted:         "encrypted-router-secret",
			MediaConverterPasswordEncrypted: "encrypted-media-secret",
			SwitchPasswordEncrypted:         "encrypted-switch-secret",
		},
	)

	if !response.ONUPasswordConfigured ||
		!response.RouterPasswordConfigured ||
		!response.MediaConverterPasswordConfigured ||
		!response.SwitchPasswordConfigured {
		t.Fatal("expected credential configured flags")
	}
	if response.MikroTikPort != "sfp-sfpplus1" {
		t.Fatalf("MikroTik port = %q", response.MikroTikPort)
	}

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	body := string(payload)

	for _, forbidden := range []string{
		"encrypted-onu-secret",
		"encrypted-router-secret",
		"encrypted-media-secret",
		"encrypted-switch-secret",
		"onu_password\"",
		"router_password\"",
		"media_converter_password\"",
		"switch_password\"",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response exposed forbidden secret field/value %q", forbidden)
		}
	}
}
