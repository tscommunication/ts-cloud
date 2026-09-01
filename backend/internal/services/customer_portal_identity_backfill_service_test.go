package services

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func TestBackfillCustomerPortalIdentitiesCreatesLoginForEncryptedImport(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:portal_identity_backfill?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}, &models.CustomerInternetAccount{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	key := "0123456789abcdef0123456789abcdef"
	password := "legacy-portal-password"
	encrypted, err := security.EncryptSecret(password, key)
	if err != nil {
		t.Fatal(err)
	}
	customer := models.Customer{CustomerCode: "IMP-PORTAL-1", FullName: "Imported Customer", Mobile: "01700000000", Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	account := models.CustomerInternetAccount{AccountCode: "NET-PORTAL-1", CustomerID: customer.ID, RouterID: 1, PPPoEUsername: "legacy-pppoe", PPPoEPasswordEncrypted: encrypted, Status: "ACTIVE"}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	result, err := BackfillCustomerPortalIdentities(key)
	if err != nil {
		t.Fatal(err)
	}
	if result.Created != 1 || result.Updated != 0 {
		t.Fatalf("unexpected backfill result: %+v", result)
	}
	var identity models.User
	if err := db.Where("customer_id = ?", customer.ID).First(&identity).Error; err != nil {
		t.Fatal(err)
	}
	if identity.Role != "customer" || !identity.Active {
		t.Fatalf("unexpected customer identity: %+v", identity)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(identity.Password), []byte(password)); err != nil {
		t.Fatalf("portal password hash does not match imported password: %v", err)
	}
}
