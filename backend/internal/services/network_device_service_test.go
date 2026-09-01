package services

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

const networkDeviceManagementCredentialTestKey = "0123456789abcdef0123456789abcdef"

func setupNetworkDeviceServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(filepath.Join(t.TempDir(), "network_device.db")),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.NetworkDevice{},
		&models.NetworkRouter{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func validManagementCredentialTestDevice() models.NetworkDevice {
	return models.NetworkDevice{
		Code:               "olt-test-01",
		Name:               "OLT Test 01",
		DeviceType:         "OLT",
		Vendor:             "ECOM",
		DeviceModel:        "Test OLT",
		OLTType:            "EPON",
		ManagementIP:       "192.0.2.10",
		ManagementPort:     8010,
		ManagementUsername: "  admin-test  ",
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		PollingInterval:    300,
		MonitoringEnabled:  true,
	}
}

func TestSaveNetworkDeviceEncryptsManagementCredential(t *testing.T) {
	db := setupNetworkDeviceServiceTestDB(t)

	row := validManagementCredentialTestDevice()

	if err := SaveNetworkDevice(
		&row,
		nil,
		"snmp-community",
		"management-password",
		networkDeviceManagementCredentialTestKey,
	); err != nil {
		t.Fatal(err)
	}

	if row.ID == 0 {
		t.Fatal("expected saved network device ID")
	}

	var saved models.NetworkDevice
	if err := db.First(&saved, row.ID).Error; err != nil {
		t.Fatal(err)
	}

	if saved.ManagementUsername != "admin-test" {
		t.Fatalf(
			"management username = %q, want %q",
			saved.ManagementUsername,
			"admin-test",
		)
	}

	if saved.ManagementSecretEncrypted == "" {
		t.Fatal("expected encrypted management credential")
	}

	if saved.ManagementSecretEncrypted == "management-password" {
		t.Fatal("management credential must not be stored as plaintext")
	}

	decrypted, err := security.DecryptSecret(
		saved.ManagementSecretEncrypted,
		networkDeviceManagementCredentialTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	if decrypted != "management-password" {
		t.Fatalf(
			"decrypted management credential = %q, want expected value",
			decrypted,
		)
	}
}

func TestSaveNetworkDeviceBlankManagementCredentialPreservesCiphertext(t *testing.T) {
	db := setupNetworkDeviceServiceTestDB(t)

	row := validManagementCredentialTestDevice()

	if err := SaveNetworkDevice(
		&row,
		nil,
		"snmp-community",
		"management-password",
		networkDeviceManagementCredentialTestKey,
	); err != nil {
		t.Fatal(err)
	}

	var existing models.NetworkDevice
	if err := db.First(&existing, row.ID).Error; err != nil {
		t.Fatal(err)
	}

	originalCiphertext := existing.ManagementSecretEncrypted
	if originalCiphertext == "" {
		t.Fatal("expected original encrypted management credential")
	}

	existing.Name = "OLT Test 01 Updated"

	if err := SaveNetworkDevice(
		&existing,
		nil,
		"",
		"",
		networkDeviceManagementCredentialTestKey,
	); err != nil {
		t.Fatal(err)
	}

	var saved models.NetworkDevice
	if err := db.First(&saved, existing.ID).Error; err != nil {
		t.Fatal(err)
	}

	if saved.ManagementSecretEncrypted != originalCiphertext {
		t.Fatal("blank management credential must preserve existing ciphertext")
	}
}

func TestSaveNetworkDeviceReplacesManagementCredential(t *testing.T) {
	db := setupNetworkDeviceServiceTestDB(t)

	row := validManagementCredentialTestDevice()

	if err := SaveNetworkDevice(
		&row,
		nil,
		"snmp-community",
		"old-management-password",
		networkDeviceManagementCredentialTestKey,
	); err != nil {
		t.Fatal(err)
	}

	var existing models.NetworkDevice
	if err := db.First(&existing, row.ID).Error; err != nil {
		t.Fatal(err)
	}

	oldCiphertext := existing.ManagementSecretEncrypted

	if err := SaveNetworkDevice(
		&existing,
		nil,
		"",
		"new-management-password",
		networkDeviceManagementCredentialTestKey,
	); err != nil {
		t.Fatal(err)
	}

	var saved models.NetworkDevice
	if err := db.First(&saved, existing.ID).Error; err != nil {
		t.Fatal(err)
	}

	if saved.ManagementSecretEncrypted == "" {
		t.Fatal("expected replacement encrypted management credential")
	}

	if saved.ManagementSecretEncrypted == oldCiphertext {
		t.Fatal("replacement credential must replace previous ciphertext")
	}

	decrypted, err := security.DecryptSecret(
		saved.ManagementSecretEncrypted,
		networkDeviceManagementCredentialTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	if decrypted != "new-management-password" {
		t.Fatalf(
			"decrypted replacement credential = %q, want expected value",
			decrypted,
		)
	}
}
