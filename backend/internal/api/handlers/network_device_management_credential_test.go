package handlers

import (
	"encoding/json"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestNetworkDeviceResponseDoesNotExposeManagementCredential(t *testing.T) {
	row := models.NetworkDevice{
		ManagementUsername:        "admin-test",
		ManagementSecretEncrypted: "encrypted-secret-must-not-leak",
	}

	response := networkDeviceResponse(row, "unused-test-key")

	if got := response["management_username"]; got != "admin-test" {
		t.Fatalf(
			"management_username = %#v, want %q",
			got,
			"admin-test",
		)
	}

	configured, ok := response["management_credential_configured"].(bool)
	if !ok || !configured {
		t.Fatalf(
			"management_credential_configured = %#v, want true",
			response["management_credential_configured"],
		)
	}

	if _, exists := response["management_secret"]; exists {
		t.Fatal("response must not expose management_secret")
	}

	if _, exists := response["management_secret_encrypted"]; exists {
		t.Fatal("response must not expose management_secret_encrypted")
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}

	if string(encoded) == "" {
		t.Fatal("expected JSON response")
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}

	if _, exists := decoded["management_secret"]; exists {
		t.Fatal("JSON response must not contain management_secret")
	}

	if _, exists := decoded["management_secret_encrypted"]; exists {
		t.Fatal("JSON response must not contain management_secret_encrypted")
	}
}
