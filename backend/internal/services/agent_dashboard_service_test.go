package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestAgentDashboardAggregatesRemainScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Agent{}, &models.Customer{}, &models.Package{}, &models.Subscription{}, &models.Invoice{}, &models.AgentCollection{}, &models.AgentSettlement{}, &models.NetworkRouter{}, &models.CustomerInternetAccount{}, &models.NetworkRouterPPPoESession{}); err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{Code: "AG-TEST", Name: "Test Agent", CommissionPercent: 10, Status: "ACTIVE"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}
	customer := models.Customer{CustomerCode: "CUS-TEST", FullName: "Scoped Customer", Mobile: "01000000000", AgentID: &agent.ID, Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{PackageCode: "PKG-TEST", Name: "Test", Price: 1000}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	subscription := models.Subscription{SubscriptionCode: "SUB-TEST", CustomerID: customer.ID, PackageID: pkg.ID, Status: "ACTIVE", ActivationDate: now, NextBillingDate: now, ExpiryDate: now.AddDate(0, 1, 0), BillingDay: 1}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}
	router := models.NetworkRouter{Code: "RTR-TEST", Name: "Test Router", Host: "10.0.0.1", APIPort: 8728, APIUsername: "read"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	account := models.CustomerInternetAccount{AccountCode: "NET-TEST", CustomerID: customer.ID, RouterID: router.ID, PackageID: pkg.ID, PPPoEUsername: "customer-test", Status: "ACTIVE", ExpiryDate: ptrTime(now.AddDate(0, 1, 0))}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	session := models.NetworkRouterPPPoESession{RouterID: router.ID, SessionKey: "session-test", Username: "customer-test", Active: true, FirstSeenAt: now, LastSeenAt: now}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	invoice := models.Invoice{InvoiceNo: "INV-TEST", CustomerID: customer.ID, SubscriptionID: subscription.ID, PackageID: pkg.ID, TotalAmount: 1000, DueAmount: 400, Status: "OVERDUE", IssueDate: now, DueDate: now}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	collection := models.AgentCollection{AgentID: agent.ID, CustomerID: customer.ID, PaymentID: 1, Amount: 600, CommissionRate: 10, CommissionAmount: 60, Status: "ACTIVE", CollectedAt: now}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatal(err)
	}

	summary, err := getAgentDashboardSummary(db, agent.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalCustomers != 1 || summary.ActiveSubscriptions != 1 || summary.OnlineCustomers != 1 || summary.OfflineCustomers != 0 || summary.ExpiredCustomers != 0 || summary.TotalInvoiced != 1000 || summary.TotalOutstanding != 400 || summary.OverdueInvoices != 1 || summary.TotalCollected != 600 || summary.CommissionPayable != 60 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
