package services

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateAgentCustomerProvisionRequest(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_provision_request_service?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	database.DB = db

	if err := db.AutoMigrate(
		&models.POP{},
		&models.Agent{},
		&models.AgentPOP{},
		&models.Package{},
		&models.NetworkRouter{},
		&models.CustomerProvisionRequest{},
	); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	pop := models.POP{
		Code:   "POP-TEST",
		Name:   "Test POP",
		Status: "ACTIVE",
	}
	if err := db.Create(&pop).Error; err != nil {
		t.Fatalf("create pop: %v", err)
	}

	agent := models.Agent{
		Code:   "AGT-TEST",
		Name:   "Test Agent",
		POPID:  pop.ID,
		Status: "ACTIVE",
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}

	pkg := models.Package{
		PackageCode: "PKG-TEST",
		Name:        "Test Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	now := time.Date(2026, 8, 18, 15, 0, 0, 0, time.UTC)

	request := &models.CustomerProvisionRequest{
		FullName:       "Test Customer",
		Mobile:         "01712345678",
		NID:            "1234567890",
		PackageID:      pkg.ID,
		PPPoEUsername:  "test-user",
		PPPoEPassword:  "secret",
		BillingDay:     18,
		ActivationDate: now,
	}

	if err := CreateAgentCustomerProvisionRequest(
		request,
		agent.ID,
		77,
		now,
	); err != nil {
		t.Fatalf("create provision request: %v", err)
	}

	if request.Status != "PENDING" {
		t.Fatalf("expected PENDING status, got %q", request.Status)
	}

	if request.Source != "AGENT" {
		t.Fatalf("expected AGENT source, got %q", request.Source)
	}

	if request.AgentID == nil || *request.AgentID != agent.ID {
		t.Fatalf("expected agent_id %d, got %#v", agent.ID, request.AgentID)
	}

	if request.POPID == nil || *request.POPID != pop.ID {
		t.Fatalf("expected pop_id %d, got %#v", pop.ID, request.POPID)
	}

	if request.RequestedByUserID != 77 {
		t.Fatalf(
			"expected requested_by_user_id 77, got %d",
			request.RequestedByUserID,
		)
	}

	if request.RequestCode == "" {
		t.Fatal("expected generated request code")
	}

	var saved models.CustomerProvisionRequest
	if err := db.First(&saved, request.ID).Error; err != nil {
		t.Fatalf("reload saved request: %v", err)
	}

	if saved.Status != "PENDING" {
		t.Fatalf("expected saved PENDING status, got %q", saved.Status)
	}
}

func TestApproveCustomerProvisionRequest(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:approve_customer_provision_request?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	database.DB = db

	if err := db.AutoMigrate(
		&models.POP{},
		&models.Agent{},
		&models.AgentPOP{},
		&models.Customer{},
		&models.Package{},
		&models.NetworkRouter{},
		&models.Subscription{},
		&models.CustomerProvisionRequest{},
	); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	pop := models.POP{
		Code:   "POP-APPROVE",
		Name:   "Approve POP",
		Status: "ACTIVE",
	}
	if err := db.Create(&pop).Error; err != nil {
		t.Fatalf("create pop: %v", err)
	}

	agent := models.Agent{
		Code:   "AGT-APPROVE",
		Name:   "Approve Agent",
		POPID:  pop.ID,
		Status: "ACTIVE",
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}

	pkg := models.Package{
		PackageCode: "PKG-APPROVE",
		Name:        "Approve Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	now := time.Date(2026, 8, 18, 15, 20, 0, 0, time.UTC)

	request := models.CustomerProvisionRequest{
		RequestCode:       "CPR-APPROVE-001",
		Source:            "AGENT",
		Status:            "PENDING",
		AgentID:           &agent.ID,
		POPID:             &pop.ID,
		FullName:          "Approved Customer",
		Mobile:            "01712345679",
		NID:               "1234567891",
		Country:           "Bangladesh",
		PackageID:         pkg.ID,
		PPPoEUsername:     "approved-user",
		PPPoEPassword:     "secret",
		BillingDay:        18,
		ActivationDate:    now,
		RequestedByUserID: 50,
		RequestedAt:       now.Add(-time.Hour),
	}

	if err := db.Create(&request).Error; err != nil {
		t.Fatalf("create provision request: %v", err)
	}

	if err := ApproveCustomerProvisionRequest(
		&request,
		99,
		now,
	); err != nil {
		t.Fatalf("approve provision request: %v", err)
	}

	if request.Status != "COMPLETED" {
		t.Fatalf("expected COMPLETED status, got %q", request.Status)
	}

	if request.CustomerID == nil || *request.CustomerID == 0 {
		t.Fatal("expected approved customer_id")
	}

	if request.SubscriptionID == nil || *request.SubscriptionID == 0 {
		t.Fatal("expected approved subscription_id")
	}

	if request.ReviewedByUserID == nil || *request.ReviewedByUserID != 99 {
		t.Fatalf(
			"expected reviewed_by_user_id 99, got %#v",
			request.ReviewedByUserID,
		)
	}

	if request.ReviewedAt == nil {
		t.Fatal("expected reviewed_at")
	}

	var customer models.Customer
	if err := db.First(&customer, *request.CustomerID).Error; err != nil {
		t.Fatalf("load approved customer: %v", err)
	}

	if customer.FullName != "Approved Customer" {
		t.Fatalf("unexpected customer name %q", customer.FullName)
	}

	if customer.AgentID == nil || *customer.AgentID != agent.ID {
		t.Fatalf("expected customer agent_id %d", agent.ID)
	}

	if customer.PopID == nil || *customer.PopID != pop.ID {
		t.Fatalf("expected customer pop_id %d", pop.ID)
	}

	var subscription models.Subscription
	if err := db.First(&subscription, *request.SubscriptionID).Error; err != nil {
		t.Fatalf("load approved subscription: %v", err)
	}

	if subscription.CustomerID != customer.ID {
		t.Fatalf(
			"expected subscription customer_id %d, got %d",
			customer.ID,
			subscription.CustomerID,
		)
	}

	if subscription.PackageID != pkg.ID {
		t.Fatalf(
			"expected package_id %d, got %d",
			pkg.ID,
			subscription.PackageID,
		)
	}

	if subscription.PPPoEUsername != "approved-user" {
		t.Fatalf(
			"unexpected PPPoE username %q",
			subscription.PPPoEUsername,
		)
	}

	var savedRequest models.CustomerProvisionRequest
	if err := db.First(&savedRequest, request.ID).Error; err != nil {
		t.Fatalf("reload provision request: %v", err)
	}

	if savedRequest.Status != "COMPLETED" {
		t.Fatalf(
			"expected saved COMPLETED status, got %q",
			savedRequest.Status,
		)
	}
}

func TestRejectCustomerProvisionRequest(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:reject_customer_provision_request?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	database.DB = db

	if err := db.AutoMigrate(
		&models.CustomerProvisionRequest{},
	); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	now := time.Date(2026, 8, 18, 15, 30, 0, 0, time.UTC)

	request := models.CustomerProvisionRequest{
		RequestCode:       "CPR-REJECT-001",
		Source:            "AGENT",
		Status:            "PENDING",
		FullName:          "Rejected Customer",
		Mobile:            "01712345680",
		NID:               "1234567892",
		PackageID:         1,
		PPPoEUsername:     "rejected-user",
		BillingDay:        18,
		ActivationDate:    now,
		RequestedByUserID: 50,
		RequestedAt:       now.Add(-time.Hour),
	}

	if err := db.Create(&request).Error; err != nil {
		t.Fatalf("create provision request: %v", err)
	}

	if err := RejectCustomerProvisionRequest(
		&request,
		99,
		"Duplicate customer information",
		now,
	); err != nil {
		t.Fatalf("reject provision request: %v", err)
	}

	if request.Status != "REJECTED" {
		t.Fatalf("expected REJECTED status, got %q", request.Status)
	}

	if request.RejectionReason != "Duplicate customer information" {
		t.Fatalf(
			"unexpected rejection reason %q",
			request.RejectionReason,
		)
	}

	if request.ReviewedByUserID == nil || *request.ReviewedByUserID != 99 {
		t.Fatalf(
			"expected reviewed_by_user_id 99, got %#v",
			request.ReviewedByUserID,
		)
	}

	if request.ReviewedAt == nil {
		t.Fatal("expected reviewed_at")
	}

	var saved models.CustomerProvisionRequest
	if err := db.First(&saved, request.ID).Error; err != nil {
		t.Fatalf("reload provision request: %v", err)
	}

	if saved.Status != "REJECTED" {
		t.Fatalf("expected saved REJECTED status, got %q", saved.Status)
	}

	if saved.RejectionReason != "Duplicate customer information" {
		t.Fatalf(
			"unexpected saved rejection reason %q",
			saved.RejectionReason,
		)
	}
}
