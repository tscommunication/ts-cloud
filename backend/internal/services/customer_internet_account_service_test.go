package services

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestSaveCustomerInternetCredentialSynchronizesPortalIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:customer_internet_credential?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}, &models.CustomerInternetAccount{}, &models.Subscription{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })

	customer := models.Customer{CustomerCode: "CUS-000001", FullName: "Internet Customer", Mobile: "01700000008", Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	identity := models.User{Name: customer.FullName, Username: customer.CustomerCode, Email: "cus1@customer.invalid", Password: "inactive", Role: "customer", Active: false, CustomerID: &customer.ID}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&identity).Update("active", false).Error; err != nil {
		t.Fatal(err)
	}

	key := "0123456789abcdef0123456789abcdef"
	macAddress := "C0:A4:76:F7:F7:DD"
	staticIPAddress := "10.9.0.220"
	account, err := SaveCustomerInternetCredential(customer.ID, CustomerInternetCredentialInput{
		RouterID: 7, PPPoEUsername: "pppoe-cus-1", PPPoEPassword: "shared-pass",
		MACAddress: &macAddress, StaticIPAddress: &staticIPAddress,
	}, key, true)
	if err != nil {
		t.Fatal(err)
	}
	if account.PPPoEUsername != "pppoe-cus-1" || account.RouterID != 7 {
		t.Fatalf("unexpected account: %+v", account)
	}
	if account.MACAddress != macAddress || account.StaticIPAddress != staticIPAddress {
		t.Fatalf("network bindings were not saved: %+v", account)
	}
	_, password, err := GetCustomerInternetCredential(customer.ID, key)
	if err != nil || password != "shared-pass" {
		t.Fatalf("credential round trip failed: password=%q err=%v", password, err)
	}
	if err := db.First(&identity, identity.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !identity.Active {
		t.Fatal("customer portal identity was not activated")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(identity.Password), []byte("shared-pass")); err != nil {
		t.Fatal("portal password was not synchronized")
	}
}

func TestGetCustomerInternetCredentialRejectsBlankCredentialClearly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:blank_customer_internet_credential?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.CustomerInternetAccount{}); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })

	customer := models.Customer{
		CustomerCode: "CUS-BLANK-001",
		FullName:     "Blank Credential Customer",
		Mobile:       "01700000009",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	account := models.CustomerInternetAccount{
		AccountCode:            "NET-BLANK-001",
		CustomerID:             customer.ID,
		RouterID:               1,
		PPPoEUsername:          "blank-user",
		PPPoEPasswordEncrypted: "",
		Status:                 "ACTIVE",
		SyncIntervalMinutes:    30,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	_, _, err = GetCustomerInternetCredential(
		customer.ID,
		"0123456789abcdef0123456789abcdef",
	)
	if err == nil {
		t.Fatal("expected blank PPPoE credential to fail")
	}
	if err.Error() != "customer PPPoE credential is not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentCredentialUpdateCannotChangeUsername(t *testing.T) {
	// The false allowIdentityEdit flag models an agent updating an existing account.
	db, err := gorm.Open(sqlite.Open("file:agent_customer_credential?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}, &models.CustomerInternetAccount{}, &models.Subscription{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	customer := models.Customer{CustomerCode: "CUS-000002", FullName: "Agent Customer", Mobile: "01700000007", Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	identity := models.User{Name: customer.FullName, Username: customer.CustomerCode, Email: "cus2@customer.invalid", Password: "inactive", Role: "customer", Active: false, CustomerID: &customer.ID}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatal(err)
	}
	account := models.CustomerInternetAccount{AccountCode: "NET-000002", CustomerID: customer.ID, RouterID: 1, PPPoEUsername: "fixed-user", PPPoEPasswordEncrypted: "old", Status: "ACTIVE"}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	_, err = SaveCustomerInternetCredential(customer.ID, CustomerInternetCredentialInput{RouterID: 1, PPPoEUsername: "changed-user", PPPoEPassword: "new-password"}, "0123456789abcdef0123456789abcdef", false)
	if err == nil {
		t.Fatal("expected agent username change to be rejected")
	}
}

func TestPPPoEPasswordUpdateSynchronizesActivePortalPassword(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:independent_portal_password?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}, &models.CustomerInternetAccount{}, &models.Subscription{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	customer := models.Customer{CustomerCode: "CUS-000003", FullName: "Independent Password", Mobile: "01700000006", Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	portalHash, err := bcrypt.GenerateFromPassword([]byte("portal-only"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	identity := models.User{Name: customer.FullName, Username: customer.CustomerCode, Email: "cus3@customer.invalid", Password: string(portalHash), Role: "customer", Active: true, CustomerID: &customer.ID}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatal(err)
	}
	key := "0123456789abcdef0123456789abcdef"
	account := models.CustomerInternetAccount{AccountCode: "NET-000003", CustomerID: customer.ID, RouterID: 1, PPPoEUsername: "independent-user", PPPoEPasswordEncrypted: "placeholder", Status: "ACTIVE"}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := SaveCustomerInternetCredential(customer.ID, CustomerInternetCredentialInput{RouterID: 1, PPPoEUsername: "independent-user", PPPoEPassword: "new-pppoe-password"}, key, false); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&identity, identity.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(identity.Password), []byte("new-pppoe-password")); err != nil {
		t.Fatal("active portal password was not synchronized with PPPoE update")
	}
}

func TestSaveCustomerInternetCredentialPreservesBlankLegacyCredential(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:preserve_blank_legacy_credential?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Customer{},
		&models.User{},
		&models.CustomerInternetAccount{},
		&models.Subscription{},
	); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })

	customer := models.Customer{
		CustomerCode: "IMP-TEST-79",
		FullName:     "Legacy Imported Customer",
		Mobile:       "01918878228",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	identity := models.User{
		Name:       customer.FullName,
		Username:   customer.CustomerCode,
		Email:      "legacy-customer@customer.invalid",
		Password:   "inactive",
		Role:       "customer",
		Active:     false,
		CustomerID: &customer.ID,
	}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatal(err)
	}

	account := models.CustomerInternetAccount{
		AccountCode:            "NET-IMP-TEST-79",
		CustomerID:             customer.ID,
		RouterID:               1,
		PPPoEUsername:          "legacy-adopted-user",
		PPPoEPasswordEncrypted: "",
		Status:                 "ACTIVE",
		SyncIntervalMinutes:    30,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	saved, err := SaveCustomerInternetCredential(
		customer.ID,
		CustomerInternetCredentialInput{
			RouterID:            1,
			PPPoEUsername:       "legacy-adopted-user",
			PPPoEPassword:       "",
			SyncIntervalMinutes: 30,
		},
		"0123456789abcdef0123456789abcdef",
		true,
	)
	if err != nil {
		t.Fatalf("existing adopted account should preserve blank credential: %v", err)
	}

	if saved.PPPoEPasswordEncrypted != "" {
		t.Fatal("blank legacy credential was unexpectedly changed")
	}
	if saved.PPPoEUsername != "legacy-adopted-user" {
		t.Fatalf("unexpected PPPoE username: %q", saved.PPPoEUsername)
	}
}

func TestSaveCustomerInternetCredentialRejectsInvalidBindings(t *testing.T) {
	invalidMAC := "not-a-mac"
	if _, err := SaveCustomerInternetCredential(1, CustomerInternetCredentialInput{
		RouterID: 1, PPPoEUsername: "binding-user", PPPoEPassword: "shared-pass",
		MACAddress: &invalidMAC,
	}, "0123456789abcdef0123456789abcdef", true); err == nil {
		t.Fatal("expected invalid MAC address to be rejected")
	}

	invalidIP := "999.1.1.1"
	if _, err := SaveCustomerInternetCredential(1, CustomerInternetCredentialInput{
		RouterID: 1, PPPoEUsername: "binding-user", PPPoEPassword: "shared-pass",
		StaticIPAddress: &invalidIP,
	}, "0123456789abcdef0123456789abcdef", true); err == nil {
		t.Fatal("expected invalid static IP address to be rejected")
	}
}
