package services

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func temporaryAccessTestDatabase(t *testing.T) (*gorm.DB, models.Customer, models.Subscription, models.CustomerInternetAccount) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.CustomerInternetAccount{},
		&models.Subscription{}, &models.TemporaryInternetAccess{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })

	customer := models.Customer{CustomerCode: "CUS-TEMP-1", FullName: "Temporary Customer", Mobile: "01700000991"}
	pkg := models.Package{PackageCode: "PKG-TEMP-1", Name: "Temporary Package", Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	expiry := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.Local)
	account := models.CustomerInternetAccount{AccountCode: "NET-TEMP-1", CustomerID: customer.ID, RouterID: 1,
		PPPoEUsername: "temp-user", PPPoEPasswordEncrypted: "encrypted", PackageID: pkg.ID,
		BillingDay: 20, ExpiryDate: &expiry, NextBillingDate: &expiry, Status: "EXPIRED"}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	subscription := models.Subscription{SubscriptionCode: "SUB-TEMP-1", CustomerID: customer.ID, PackageID: pkg.ID,
		InternetAccountID: &account.ID, ActivationDate: expiry.AddDate(0, -1, 0), BillingDay: 20,
		NextBillingDate: expiry, ExpiryDate: expiry, Status: "EXPIRED"}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}
	return db, customer, subscription, account
}

func TestGrantTemporaryInternetAccessEnforcesOneToSevenDaysAndCreatesDeduction(t *testing.T) {
	db, customer, subscription, account := temporaryAccessTestDatabase(t)
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.Local)
	for _, days := range []int{0, 8} {
		_, err := GrantTemporaryInternetAccess(TemporaryInternetAccessGrantInput{CustomerID: customer.ID, Days: days,
			RequestSource: "CUSTOMER", Reason: "promise", GrantedByUserID: 1, Now: now})
		if err == nil {
			t.Fatalf("expected %d days to be rejected", days)
		}
	}

	grant, err := GrantTemporaryInternetAccess(TemporaryInternetAccessGrantInput{CustomerID: customer.ID, Days: 2,
		RequestSource: "customer", Reason: "pay after two days", GrantedByUserID: 1, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if grant.SubscriptionID != subscription.ID || grant.GrantedDurationSeconds != 2*24*60*60 ||
		!grant.EndsAt.Equal(now.Add(48*time.Hour)) {
		t.Fatalf("unexpected grant: %+v", grant)
	}
	if err := db.First(&account, account.ID).Error; err != nil {
		t.Fatal(err)
	}
	if account.Status != TemporaryInternetStatusActive {
		t.Fatalf("status = %q", account.Status)
	}
	deduction, items, err := PendingTemporaryAccessDeductionTx(db, subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deduction != 48*time.Hour || len(items) != 1 {
		t.Fatalf("deduction=%v items=%d", deduction, len(items))
	}
}

func TestExpireDueTemporaryInternetAccessRestoresExpiredInternetStatus(t *testing.T) {
	db, customer, _, account := temporaryAccessTestDatabase(t)
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.Local)
	grant, err := GrantTemporaryInternetAccess(TemporaryInternetAccessGrantInput{CustomerID: customer.ID, Days: 1,
		RequestSource: "RESELLER", Reason: "one day promise", GrantedByUserID: 2, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	due, err := ExpireDueTemporaryInternetAccess(grant.EndsAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].Status != TemporaryInternetAccessExpired {
		t.Fatalf("unexpected due: %+v", due)
	}
	if err := db.First(&account, account.ID).Error; err != nil {
		t.Fatal(err)
	}
	if account.Status != "EXPIRED" {
		t.Fatalf("status = %q", account.Status)
	}
}
