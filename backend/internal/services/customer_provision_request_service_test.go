package services

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const provisionRequestCredentialKey = "0123456789abcdef0123456789abcdef"

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

	if err := SetProvisionRequestPPPoEPassword(
		request,
		request.PPPoEPassword,
		provisionRequestCredentialKey,
	); err != nil {
		t.Fatalf("encrypt provision PPPoE password: %v", err)
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

	credentialKey := "0123456789abcdef0123456789abcdef"

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
		credentialKey,
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

	if subscription.PPPoEPassword != "" {
		t.Fatalf(
			"expected plaintext PPPoE password to be cleared, got length %d",
			len(subscription.PPPoEPassword),
		)
	}

	if subscription.PPPoEPasswordEncrypted == "" {
		t.Fatal("expected encrypted PPPoE password on subscription")
	}

	decryptedPassword, err := security.DecryptSecret(
		subscription.PPPoEPasswordEncrypted,
		credentialKey,
	)
	if err != nil {
		t.Fatalf(
			"decrypt subscription PPPoE password: %v",
			err,
		)
	}

	if decryptedPassword != "secret" {
		t.Fatalf(
			"unexpected decrypted PPPoE password %q",
			decryptedPassword,
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

	if savedRequest.PPPoEPassword != "" {
		t.Fatalf(
			"expected provision plaintext PPPoE password to be cleared, got length %d",
			len(savedRequest.PPPoEPassword),
		)
	}

	if savedRequest.PPPoEPasswordEncrypted == "" {
		t.Fatal(
			"expected provision request PPPoE password to be encrypted",
		)
	}

	requestPassword, err := security.DecryptSecret(
		savedRequest.PPPoEPasswordEncrypted,
		credentialKey,
	)
	if err != nil {
		t.Fatalf(
			"decrypt provision request PPPoE password: %v",
			err,
		)
	}

	if requestPassword != "secret" {
		t.Fatalf(
			"unexpected provision request decrypted password %q",
			requestPassword,
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

func TestCreateAgentCustomerProvisionRequestRejectsRouterOutsideAgentScope(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:provision_request_router_scope_reject?mode=memory&cache=shared"),
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

	agentPOP := models.POP{
		Code:   "POP-SCOPE-A",
		Name:   "Agent POP",
		Status: "ACTIVE",
	}
	if err := db.Create(&agentPOP).Error; err != nil {
		t.Fatalf("create agent pop: %v", err)
	}

	outsidePOP := models.POP{
		Code:   "POP-SCOPE-B",
		Name:   "Outside POP",
		Status: "ACTIVE",
	}
	if err := db.Create(&outsidePOP).Error; err != nil {
		t.Fatalf("create outside pop: %v", err)
	}

	agent := models.Agent{
		Code:   "AGT-SCOPE",
		Name:   "Scoped Agent",
		POPID:  agentPOP.ID,
		Status: "ACTIVE",
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}

	pkg := models.Package{
		PackageCode: "PKG-SCOPE",
		Name:        "Scope Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	router := models.NetworkRouter{
		Code:        "RTR-OUTSIDE",
		Name:        "Outside Router",
		POPID:       &outsidePOP.ID,
		Host:        "192.0.2.50",
		APIPort:     8728,
		APIUsername: "reader",
		Status:      "ACTIVE",
	}
	if err := db.Create(&router).Error; err != nil {
		t.Fatalf("create router: %v", err)
	}

	now := time.Date(2026, 8, 18, 16, 40, 0, 0, time.UTC)

	request := &models.CustomerProvisionRequest{
		FullName:       "Outside Scope Customer",
		Mobile:         "01712345681",
		NID:            "1234567893",
		PackageID:      pkg.ID,
		RouterID:       router.ID,
		PPPoEUsername:  "outside-scope-user",
		PPPoEPassword:  "secret",
		BillingDay:     18,
		ActivationDate: now,
	}

	if err := SetProvisionRequestPPPoEPassword(
		request,
		request.PPPoEPassword,
		provisionRequestCredentialKey,
	); err != nil {
		t.Fatalf("encrypt provision PPPoE password: %v", err)
	}

	err = CreateAgentCustomerProvisionRequest(
		request,
		agent.ID,
		77,
		now,
	)
	if err == nil {
		t.Fatal("expected outside POP router to be rejected")
	}

	if err.Error() != "router is outside the agent POP scope" {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int64
	if err := db.Model(&models.CustomerProvisionRequest{}).
		Where("mobile = ?", request.Mobile).
		Count(&count).Error; err != nil {
		t.Fatalf("count provision requests: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected no saved request, got %d", count)
	}
}

func TestCreateAgentCustomerProvisionRequestUsesAdditionalRouterPOP(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:provision_request_additional_pop?mode=memory&cache=shared"),
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

	primaryPOP := models.POP{
		Code:   "POP-PRIMARY",
		Name:   "Primary POP",
		Status: "ACTIVE",
	}
	if err := db.Create(&primaryPOP).Error; err != nil {
		t.Fatalf("create primary pop: %v", err)
	}

	additionalPOP := models.POP{
		Code:   "POP-ADDITIONAL",
		Name:   "Additional POP",
		Status: "ACTIVE",
	}
	if err := db.Create(&additionalPOP).Error; err != nil {
		t.Fatalf("create additional pop: %v", err)
	}

	agent := models.Agent{
		Code:   "AGT-ADDITIONAL",
		Name:   "Additional POP Agent",
		POPID:  primaryPOP.ID,
		Status: "ACTIVE",
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}

	if err := db.Create(&models.AgentPOP{
		AgentID: agent.ID,
		POPID:   additionalPOP.ID,
	}).Error; err != nil {
		t.Fatalf("create agent-pop link: %v", err)
	}

	pkg := models.Package{
		PackageCode: "PKG-ADDITIONAL",
		Name:        "Additional Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	router := models.NetworkRouter{
		Code:        "RTR-ADDITIONAL",
		Name:        "Additional Router",
		POPID:       &additionalPOP.ID,
		Host:        "192.0.2.60",
		APIPort:     8728,
		APIUsername: "reader",
		Status:      "ACTIVE",
	}
	if err := db.Create(&router).Error; err != nil {
		t.Fatalf("create router: %v", err)
	}

	now := time.Date(2026, 8, 18, 16, 45, 0, 0, time.UTC)

	request := &models.CustomerProvisionRequest{
		FullName:       "Additional POP Customer",
		Mobile:         "01712345682",
		NID:            "1234567894",
		PackageID:      pkg.ID,
		RouterID:       router.ID,
		PPPoEUsername:  "additional-pop-user",
		PPPoEPassword:  "secret",
		BillingDay:     18,
		ActivationDate: now,
	}

	if err := SetProvisionRequestPPPoEPassword(
		request,
		request.PPPoEPassword,
		provisionRequestCredentialKey,
	); err != nil {
		t.Fatalf("encrypt provision PPPoE password: %v", err)
	}

	if err := CreateAgentCustomerProvisionRequest(
		request,
		agent.ID,
		77,
		now,
	); err != nil {
		t.Fatalf("create provision request: %v", err)
	}

	if request.POPID == nil {
		t.Fatal("expected request pop_id")
	}

	if *request.POPID != additionalPOP.ID {
		t.Fatalf(
			"expected additional pop_id %d, got %d",
			additionalPOP.ID,
			*request.POPID,
		)
	}
}
