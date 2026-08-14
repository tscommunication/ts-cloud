package services

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMapNetworkPPPoESessionPreservesSubscriptionPasswordAndRejectsDuplicate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:pppoe_mapping_service?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&models.NetworkRouter{}, &models.Customer{}, &models.Package{}, &models.Subscription{}, &models.NetworkRouterPPPoESession{}); err != nil {
		t.Fatal(err)
	}
	router := models.NetworkRouter{Code: "MAP-R1", Name: "Mapping Router", Host: "10.20.0.1", APIPort: 8729, APIUsername: "reader", Status: "ACTIVE"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	customer := models.Customer{CustomerCode: "CUS-MAP", FullName: "Mapping Customer", Mobile: "01010000000"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{PackageCode: "PKG-MAP", Name: "Mapping Package", Status: "ACTIVE"}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	subscription := models.Subscription{SubscriptionCode: "SUB-MAP", CustomerID: customer.ID, PackageID: pkg.ID, Status: "ACTIVE", PPPoEPassword: "keep-this-secret"}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}
	session := models.NetworkRouterPPPoESession{RouterID: router.ID, SessionKey: "live", Username: "live-user", Active: true, FirstSeenAt: time.Now(), LastSeenAt: time.Now()}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	if err := MapNetworkPPPoESession(session.ID, subscription.ID); err != nil {
		t.Fatal(err)
	}
	var mapped models.Subscription
	if err := db.First(&mapped, subscription.ID).Error; err != nil {
		t.Fatal(err)
	}
	if mapped.RouterID != router.ID || mapped.PPPoEUsername != "live-user" || mapped.PPPoEPassword != "keep-this-secret" {
		t.Fatalf("unexpected mapped subscription: %+v", mapped)
	}
	other := models.Subscription{SubscriptionCode: "SUB-OTHER", CustomerID: customer.ID, PackageID: pkg.ID, Status: "ACTIVE"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if err := MapNetworkPPPoESession(session.ID, other.ID); err == nil {
		t.Fatal("expected duplicate PPPoE username mapping to be rejected")
	}
}
