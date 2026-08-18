package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

func CreateAgentCustomerProvisionRequest(
	request *models.CustomerProvisionRequest,
	agentID uint,
	requestedByUserID uint,
	now time.Time,
) error {
	if agentID == 0 {
		return fmt.Errorf("agent account is not linked")
	}

	agent, err := repositories.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("agent not found")
	}
	if agent.Status != "ACTIVE" {
		return fmt.Errorf("agent must be active")
	}

	request.FullName = strings.TrimSpace(request.FullName)
	request.Mobile = strings.TrimSpace(request.Mobile)
	request.AltMobile = strings.TrimSpace(request.AltMobile)
	request.NID = strings.TrimSpace(request.NID)
	request.Email = strings.TrimSpace(request.Email)
	request.PPPoEUsername = strings.TrimSpace(request.PPPoEUsername)
	request.PPPoEPassword = strings.TrimSpace(request.PPPoEPassword)
	request.Remarks = strings.TrimSpace(request.Remarks)

	if request.FullName == "" {
		return fmt.Errorf("full name is required")
	}

	if err := ValidateCustomerIdentity(
		request.Mobile,
		request.AltMobile,
		request.NID,
	); err != nil {
		return err
	}

	if request.PackageID == 0 {
		return fmt.Errorf("package is required")
	}

	pkg, err := repositories.GetPackageByID(request.PackageID)
	if err != nil {
		return fmt.Errorf("package not found")
	}
	if pkg.Status != "ACTIVE" {
		return fmt.Errorf("package must be active")
	}

	popID, err := resolveAgentProvisionPOP(
		agentID,
		agent.POPID,
		request.RouterID,
	)
	if err != nil {
		return err
	}

	if request.PPPoEUsername == "" {
		return fmt.Errorf("PPPoE username is required")
	}

	if request.BillingDay < 1 || request.BillingDay > 31 {
		return fmt.Errorf("billing day must be between 1 and 31")
	}

	if request.ActivationDate.IsZero() {
		request.ActivationDate = now
	}

	requestCode := "CPR-000001"
	lastRequest, err := repositories.GetLastCustomerProvisionRequest()
	if err == nil {
		requestCode = fmt.Sprintf("CPR-%06d", lastRequest.ID+1)
	}

	request.RequestCode = requestCode
	request.Source = "AGENT"
	request.Status = "PENDING"
	request.AgentID = &agentID
	if popID > 0 {
		request.POPID = &popID
	}
	request.RequestedByUserID = requestedByUserID
	request.RequestedAt = now
	request.ReviewedByUserID = nil
	request.ReviewedAt = nil
	request.RejectionReason = ""
	request.CustomerID = nil
	request.SubscriptionID = nil

	return repositories.CreateCustomerProvisionRequest(request)
}

func resolveAgentProvisionPOP(
	agentID uint,
	primaryPOPID uint,
	routerID uint,
) (uint, error) {
	if agentID == 0 {
		return 0, fmt.Errorf("agent account is not linked")
	}

	if routerID == 0 {
		if primaryPOPID == 0 {
			return 0, fmt.Errorf("agent does not have a primary POP")
		}
		return primaryPOPID, nil
	}

	router, err := repositories.GetNetworkRouter(routerID)
	if err != nil {
		return 0, fmt.Errorf("router not found")
	}

	if router.Status != "ACTIVE" {
		return 0, fmt.Errorf("subscription requires an active router")
	}

	if router.POPID == nil || *router.POPID == 0 {
		return 0, fmt.Errorf("router is not assigned to a POP")
	}

	allowed, err := repositories.AgentHasPOP(agentID, *router.POPID)
	if err != nil {
		return 0, fmt.Errorf("failed to validate agent router scope")
	}
	if !allowed {
		return 0, fmt.Errorf("router is outside the agent POP scope")
	}

	return *router.POPID, nil
}

func GetCustomerProvisionRequestByID(id uint) (*models.CustomerProvisionRequest, error) {
	return repositories.GetCustomerProvisionRequestByID(id)
}

func ListCustomerProvisionRequests(
	status string,
	agentID uint,
) ([]models.CustomerProvisionRequest, error) {
	status = strings.ToUpper(strings.TrimSpace(status))

	switch status {
	case "", "PENDING", "APPROVED", "REJECTED", "CANCELLED", "COMPLETED":
	default:
		return nil, fmt.Errorf("invalid provision request status")
	}

	return repositories.ListCustomerProvisionRequests(
		repositories.CustomerProvisionRequestListParams{
			Status:  status,
			AgentID: agentID,
		},
	)
}

func RejectCustomerProvisionRequest(
	request *models.CustomerProvisionRequest,
	reviewedByUserID uint,
	reason string,
	now time.Time,
) error {
	if request == nil {
		return fmt.Errorf("provision request is required")
	}

	if request.Status != "PENDING" {
		return fmt.Errorf("only pending provision requests can be rejected")
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("rejection reason is required")
	}

	request.Status = "REJECTED"
	request.RejectionReason = reason
	request.ReviewedByUserID = &reviewedByUserID
	request.ReviewedAt = &now

	return repositories.UpdateCustomerProvisionRequest(request)
}

func ApproveCustomerProvisionRequest(
	request *models.CustomerProvisionRequest,
	reviewedByUserID uint,
	now time.Time,
) error {
	if request == nil {
		return fmt.Errorf("provision request is required")
	}

	if request.Status != "PENDING" {
		return fmt.Errorf("only pending provision requests can be approved")
	}

	if err := ValidateCustomerIdentity(
		request.Mobile,
		request.AltMobile,
		request.NID,
	); err != nil {
		return err
	}

	pkg, err := repositories.GetPackageByID(request.PackageID)
	if err != nil {
		return fmt.Errorf("package not found")
	}
	if pkg.Status != "ACTIVE" {
		return fmt.Errorf("package must be active")
	}

	if request.AgentID == nil || *request.AgentID == 0 {
		return fmt.Errorf("provision request requires an agent")
	}

	agent, err := repositories.GetAgent(*request.AgentID)
	if err != nil {
		return fmt.Errorf("agent not found")
	}

	expectedPOP, err := resolveAgentProvisionPOP(
		*request.AgentID,
		agent.POPID,
		request.RouterID,
	)
	if err != nil {
		return err
	}

	if request.POPID == nil || *request.POPID != expectedPOP {
		return fmt.Errorf("provision request POP does not match agent/router scope")
	}

	if strings.TrimSpace(request.PPPoEUsername) == "" {
		return fmt.Errorf("PPPoE username is required")
	}

	if request.AgentID == nil || *request.AgentID == 0 {
		return fmt.Errorf("agent is required")
	}

	if request.POPID == nil || *request.POPID == 0 {
		return fmt.Errorf("POP is required")
	}

	customerCode := "CUS-000001"
	lastCustomer, err := repositories.GetLastCustomer()
	if err == nil {
		customerCode = fmt.Sprintf("CUS-%06d", lastCustomer.ID+1)
	}

	subscriptionCode := "SUB-000001"
	lastSubscription, err := repositories.GetLastSubscription()
	if err == nil {
		subscriptionCode = fmt.Sprintf("SUB-%06d", lastSubscription.ID+1)
	}

	customer := models.Customer{
		CustomerCode:     customerCode,
		FullName:         strings.TrimSpace(request.FullName),
		Mobile:           strings.TrimSpace(request.Mobile),
		FatherName:       strings.TrimSpace(request.FatherName),
		MotherName:       strings.TrimSpace(request.MotherName),
		AltMobile:        strings.TrimSpace(request.AltMobile),
		Email:            strings.TrimSpace(request.Email),
		NID:              strings.TrimSpace(request.NID),
		Country:          strings.TrimSpace(request.Country),
		Division:         strings.TrimSpace(request.Division),
		District:         strings.TrimSpace(request.District),
		Upazila:          strings.TrimSpace(request.Upazila),
		PostOffice:       strings.TrimSpace(request.PostOffice),
		PostalCode:       strings.TrimSpace(request.PostalCode),
		RoadOrArea:       strings.TrimSpace(request.RoadOrArea),
		VillageOrHolding: strings.TrimSpace(request.VillageOrHolding),
		BillingDay:       request.BillingDay,
		PopID:            request.POPID,
		AgentID:          request.AgentID,
		Status:           "ACTIVE",
	}

	if customer.Country == "" {
		customer.Country = "Bangladesh"
	}

	activationDate := request.ActivationDate
	if activationDate.IsZero() {
		activationDate = now
	}

	nextBillingDate := activationDate.AddDate(0, 1, 0)

	subscription := models.Subscription{
		SubscriptionCode: subscriptionCode,
		PackageID:        request.PackageID,
		ActivationDate:   activationDate,
		BillingDay:       request.BillingDay,
		NextBillingDate:  nextBillingDate,
		ExpiryDate:       nextBillingDate,
		Status:           "ACTIVE",
		RouterID:         request.RouterID,
		PPPoEUsername:    strings.TrimSpace(request.PPPoEUsername),
		PPPoEPassword:    strings.TrimSpace(request.PPPoEPassword),
		Remarks:          strings.TrimSpace(request.Remarks),
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&customer).Error; err != nil {
			return err
		}

		subscription.CustomerID = customer.ID

		if err := tx.Create(&subscription).Error; err != nil {
			return err
		}

		request.Status = "COMPLETED"
		request.ReviewedByUserID = &reviewedByUserID
		request.ReviewedAt = &now
		request.RejectionReason = ""
		request.CustomerID = &customer.ID
		request.SubscriptionID = &subscription.ID

		if err := tx.Save(request).Error; err != nil {
			return err
		}

		return nil
	})
}
