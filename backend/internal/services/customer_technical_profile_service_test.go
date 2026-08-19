package services

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func TestSaveCustomerTechnicalProfileEncryptsAndPreservesSecrets(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_technical_profile_service?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	database.DB = db

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.CustomerTechnicalProfile{},
	); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	customer := models.Customer{
		CustomerCode: "CUS-TECH-001",
		FullName:     "Technical Test",
		Mobile:       "01712345679",
		NID:          "1234567890124",
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	const key = "0123456789abcdef0123456789abcdef"

	first, err := SaveCustomerTechnicalProfile(
		customer.ID,
		CustomerTechnicalProfileInput{
			ONUMAC:                 "AA:BB:CC:DD:EE:FF",
			ONUPassword:            "onu-secret",
			RouterPassword:         "router-secret",
			MediaConverterPassword: "media-secret",
			SwitchPassword:         "switch-secret",
		},
		key,
	)
	if err != nil {
		t.Fatalf("save first technical profile: %v", err)
	}

	if first.ONUPasswordEncrypted == "" ||
		first.RouterPasswordEncrypted == "" ||
		first.MediaConverterPasswordEncrypted == "" ||
		first.SwitchPasswordEncrypted == "" {
		t.Fatal("expected all supplied passwords to be encrypted")
	}

	if first.ONUPasswordEncrypted == "onu-secret" {
		t.Fatal("ONU password was stored as plaintext")
	}

	if first.RouterPasswordEncrypted == "router-secret" {
		t.Fatal("router password was stored as plaintext")
	}

	onuPassword, err := security.DecryptSecret(
		first.ONUPasswordEncrypted,
		key,
	)
	if err != nil {
		t.Fatalf("decrypt ONU password: %v", err)
	}
	if onuPassword != "onu-secret" {
		t.Fatalf("ONU password = %q, want %q", onuPassword, "onu-secret")
	}

	routerPassword, err := security.DecryptSecret(
		first.RouterPasswordEncrypted,
		key,
	)
	if err != nil {
		t.Fatalf("decrypt router password: %v", err)
	}
	if routerPassword != "router-secret" {
		t.Fatalf(
			"router password = %q, want %q",
			routerPassword,
			"router-secret",
		)
	}

	firstID := first.ID
	originalONUPassword := first.ONUPasswordEncrypted
	originalRouterPassword := first.RouterPasswordEncrypted
	originalMediaPassword := first.MediaConverterPasswordEncrypted
	originalSwitchPassword := first.SwitchPasswordEncrypted

	second, err := SaveCustomerTechnicalProfile(
		customer.ID,
		CustomerTechnicalProfileInput{
			ONUMAC:      "11:22:33:44:55:66",
			ONUPassword: "",
		},
		key,
	)
	if err != nil {
		t.Fatalf("update technical profile: %v", err)
	}

	if second.ID != firstID {
		t.Fatalf(
			"profile ID changed from %d to %d",
			firstID,
			second.ID,
		)
	}

	if second.ONUMAC != "11:22:33:44:55:66" {
		t.Fatalf("expected updated ONU MAC, got %q", second.ONUMAC)
	}

	if second.ONUPasswordEncrypted != originalONUPassword {
		t.Fatal("blank ONU password should preserve existing encrypted value")
	}

	if second.RouterPasswordEncrypted != originalRouterPassword {
		t.Fatal("blank router password should preserve existing encrypted value")
	}

	if second.MediaConverterPasswordEncrypted != originalMediaPassword {
		t.Fatal("blank media converter password should preserve existing encrypted value")
	}

	if second.SwitchPasswordEncrypted != originalSwitchPassword {
		t.Fatal("blank switch password should preserve existing encrypted value")
	}

	var count int64
	if err := db.Model(&models.CustomerTechnicalProfile{}).
		Where("customer_id = ?", customer.ID).
		Count(&count).Error; err != nil {
		t.Fatalf("count technical profiles: %v", err)
	}

	if count != 1 {
		t.Fatalf("technical profile count = %d, want 1", count)
	}
}
