package services

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupServiceEntitlementTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open("file:"+name+"?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.ServiceEntitlement{},
	); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
	})

	return db
}

func createServiceEntitlementTestCustomer(t *testing.T, db *gorm.DB, code string) models.Customer {
	t.Helper()

	customer := models.Customer{
		CustomerCode: code,
		FullName:     "Service Entitlement Customer",
		Mobile:       "017" + code[len(code)-7:],
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	return customer
}

func TestSaveServiceEntitlementEncryptsPasswordAndNormalizesFields(t *testing.T) {
	db := setupServiceEntitlementTestDB(t, "service_entitlement_create")
	customer := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-001")

	key := "0123456789abcdef0123456789abcdef"

	row := models.ServiceEntitlement{
		CustomerID:  customer.ID,
		ServiceType: " jellyfin ",
		ServiceName: " Jellyfin Home ",
		Username:    " media-user ",
		Endpoint:    " https://media.example.test ",
		Status:      " active ",
		QuotaGB:     100,
	}

	if err := SaveServiceEntitlement(&row, "service-secret", key); err != nil {
		t.Fatal(err)
	}

	if row.ID == 0 {
		t.Fatal("expected entitlement to be created")
	}
	if row.ServiceType != "JELLYFIN" {
		t.Fatalf("expected normalized service type JELLYFIN, got %q", row.ServiceType)
	}
	if row.Status != "ACTIVE" {
		t.Fatalf("expected normalized status ACTIVE, got %q", row.Status)
	}
	if row.ServiceName != "Jellyfin Home" ||
		row.Username != "media-user" ||
		row.Endpoint != "https://media.example.test" {
		t.Fatalf("expected trimmed fields, got %+v", row)
	}
	if row.PasswordEncrypted == "" {
		t.Fatal("expected password to be encrypted")
	}
	if row.PasswordEncrypted == "service-secret" {
		t.Fatal("password must not be stored as plaintext")
	}

	password, err := DecryptServiceEntitlementPassword(&row, key)
	if err != nil {
		t.Fatal(err)
	}
	if password != "service-secret" {
		t.Fatalf("expected decrypted password service-secret, got %q", password)
	}

	var stored models.ServiceEntitlement
	if err := db.First(&stored, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.PasswordEncrypted == "" || stored.PasswordEncrypted == "service-secret" {
		t.Fatal("stored password is not safely encrypted")
	}
}

func TestSaveServiceEntitlementRequiresPasswordWhenUsernameConfigured(t *testing.T) {
	db := setupServiceEntitlementTestDB(t, "service_entitlement_password_required")
	customer := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-002")

	row := models.ServiceEntitlement{
		CustomerID:  customer.ID,
		ServiceType: "IPTV",
		ServiceName: "IPTV Account",
		Username:    "iptv-user",
		Status:      "ACTIVE",
	}

	err := SaveServiceEntitlement(
		&row,
		"",
		"0123456789abcdef0123456789abcdef",
	)
	if err == nil {
		t.Fatal("expected missing password to be rejected")
	}
	if !strings.Contains(err.Error(), "password is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.ID != 0 {
		t.Fatal("invalid entitlement must not be created")
	}
}

func TestSaveServiceEntitlementRejectsInvalidTypeAndStatus(t *testing.T) {
	db := setupServiceEntitlementTestDB(t, "service_entitlement_validation")
	customer := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-003")
	key := "0123456789abcdef0123456789abcdef"

	invalidType := models.ServiceEntitlement{
		CustomerID:  customer.ID,
		ServiceType: "UNKNOWN",
		ServiceName: "Unknown Service",
		Status:      "ACTIVE",
	}
	if err := SaveServiceEntitlement(&invalidType, "", key); err == nil {
		t.Fatal("expected invalid service type to be rejected")
	}

	invalidStatus := models.ServiceEntitlement{
		CustomerID:  customer.ID,
		ServiceType: "CLOUD_STORAGE",
		ServiceName: "Cloud Storage",
		Status:      "UNKNOWN",
	}
	if err := SaveServiceEntitlement(&invalidStatus, "", key); err == nil {
		t.Fatal("expected invalid entitlement status to be rejected")
	}
}

func TestSaveServiceEntitlementRejectsSubscriptionFromAnotherCustomer(t *testing.T) {
	db := setupServiceEntitlementTestDB(t, "service_entitlement_subscription_scope")

	customerA := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-004")
	customerB := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-005")

	subscription := models.Subscription{
		CustomerID: customerB.ID,
		Status:     "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	row := models.ServiceEntitlement{
		CustomerID:     customerA.ID,
		SubscriptionID: &subscription.ID,
		ServiceType:    "IPTV",
		ServiceName:    "Scoped IPTV",
		Status:         "ACTIVE",
	}

	err := SaveServiceEntitlement(
		&row,
		"",
		"0123456789abcdef0123456789abcdef",
	)
	if err == nil {
		t.Fatal("expected cross-customer subscription to be rejected")
	}
	if err.Error() != "subscription does not belong to customer" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateServiceEntitlementBlankPasswordPreservesCredential(t *testing.T) {
	db := setupServiceEntitlementTestDB(t, "service_entitlement_password_preserve")
	customer := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-006")
	key := "0123456789abcdef0123456789abcdef"

	row := models.ServiceEntitlement{
		CustomerID:  customer.ID,
		ServiceType: "JELLYFIN",
		ServiceName: "Original Service",
		Username:    "media-user",
		Status:      "ACTIVE",
	}

	if err := SaveServiceEntitlement(&row, "original-secret", key); err != nil {
		t.Fatal(err)
	}

	originalEncrypted := row.PasswordEncrypted
	if originalEncrypted == "" {
		t.Fatal("expected initial encrypted password")
	}

	row.ServiceName = "Updated Service"
	row.Status = "SUSPENDED"

	if err := SaveServiceEntitlement(&row, "", key); err != nil {
		t.Fatal(err)
	}

	if row.PasswordEncrypted != originalEncrypted {
		t.Fatal("blank password update must preserve existing encrypted credential")
	}

	password, err := DecryptServiceEntitlementPassword(&row, key)
	if err != nil {
		t.Fatal(err)
	}
	if password != "original-secret" {
		t.Fatalf("expected original password to remain, got %q", password)
	}

	var stored models.ServiceEntitlement
	if err := db.First(&stored, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ServiceName != "Updated Service" || stored.Status != "SUSPENDED" {
		t.Fatalf("updated fields were not persisted: %+v", stored)
	}
	if stored.PasswordEncrypted != originalEncrypted {
		t.Fatal("stored credential changed during blank-password update")
	}
}

func TestListServiceEntitlementsScopesCustomerAndDeleteRemovesRow(t *testing.T) {
	db := setupServiceEntitlementTestDB(t, "service_entitlement_list_delete")

	customerA := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-007")
	customerB := createServiceEntitlementTestCustomer(t, db, "CUS-SVC-008")
	key := "0123456789abcdef0123456789abcdef"

	first := models.ServiceEntitlement{
		CustomerID:  customerA.ID,
		ServiceType: "IPTV",
		ServiceName: "Customer A IPTV",
		Status:      "ACTIVE",
	}
	second := models.ServiceEntitlement{
		CustomerID:  customerB.ID,
		ServiceType: "CLOUD_STORAGE",
		ServiceName: "Customer B Storage",
		Status:      "ACTIVE",
	}

	if err := SaveServiceEntitlement(&first, "", key); err != nil {
		t.Fatal(err)
	}
	if err := SaveServiceEntitlement(&second, "", key); err != nil {
		t.Fatal(err)
	}

	rows, err := ListServiceEntitlements(customerA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != first.ID {
		t.Fatalf("customer-scoped list returned unexpected rows: %+v", rows)
	}

	found, err := GetServiceEntitlement(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != first.ID || found.CustomerID != customerA.ID {
		t.Fatalf("unexpected entitlement returned: %+v", found)
	}

	if err := DeleteServiceEntitlement(first.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := GetServiceEntitlement(first.ID); err == nil {
		t.Fatal("expected deleted entitlement to be unavailable")
	}

	rows, err = ListServiceEntitlements(customerA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no remaining customer A entitlements, got %+v", rows)
	}
}

func TestSaveServiceEntitlementAcceptsFTP(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:service-entitlement-ftp?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previousDB
	})

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.ServiceEntitlement{},
	); err != nil {
		t.Fatal(err)
	}

	customer := models.Customer{
		CustomerCode: "CUS-FTP-001",
		FullName:     "FTP Customer",
		Mobile:       "01000000000",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	row := models.ServiceEntitlement{
		CustomerID:  customer.ID,
		ServiceType: "ftp",
		ServiceName: "Primary FTP",
		Status:      "active",
		QuotaGB:     10,
	}

	if err := SaveServiceEntitlement(&row, "", ""); err != nil {
		t.Fatalf("SaveServiceEntitlement FTP: %v", err)
	}

	if row.ServiceType != "FTP" {
		t.Fatalf("service type = %q, want FTP", row.ServiceType)
	}
	if row.Status != "ACTIVE" {
		t.Fatalf("status = %q, want ACTIVE", row.Status)
	}
}
