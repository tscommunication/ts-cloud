package services

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestCreateCustomerCreatesPermanentInactiveIdentityAtomically(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create_customer_identity?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}, &models.Notification{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	customer := models.Customer{FullName: "Unified Customer", Mobile: "01700000001", NID: "1234567890", Status: "ACTIVE"}
	if err := CreateCustomer(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.CustomerCode != "CUS-000001" {
		t.Fatalf("expected permanent CID CUS-000001, got %q", customer.CustomerCode)
	}

	var identity models.User
	if err := db.Where("customer_id = ?", customer.ID).First(&identity).Error; err != nil {
		t.Fatal(err)
	}
	if identity.Username != customer.CustomerCode || identity.Role != "customer" || identity.Active {
		t.Fatalf("unexpected customer identity: %+v", identity)
	}
	if !strings.HasSuffix(identity.Email, "@customer.invalid") {
		t.Fatalf("unexpected internal identity email %q", identity.Email)
	}
	var notification models.Notification
	if err := db.Where("dedup_key = ?", "customer-created:1").First(&notification).Error; err != nil {
		t.Fatal(err)
	}
	if notification.EntityID != customer.ID || notification.Type != "CUSTOMER_CREATED" {
		t.Fatalf("unexpected customer notification: %+v", notification)
	}
}

func TestCreateCustomerRollsBackWhenCIDUsernameConflicts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create_customer_identity_conflict?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}, &models.Notification{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.User{Name: "Staff", Username: "CUS-000001", Email: "staff@example.com", Password: "hash", Role: "admin", Active: true}).Error; err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	customer := models.Customer{FullName: "Conflict", Mobile: "01700000002", NID: "1234567891", Status: "ACTIVE"}
	if err := CreateCustomer(&customer); err == nil {
		t.Fatal("expected CID username conflict")
	}
	var count int64
	if err := db.Model(&models.Customer{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected transaction rollback, found %d customer(s)", count)
	}
}
