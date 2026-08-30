package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

var customerChangeTypes = map[string]bool{"BILLING_CYCLE": true, "PACKAGE": true, "LINE_SHIFT": true, "CLOSE": true}

func CreateCustomerChangeRequest(row *models.CustomerChangeRequest, agentID, userID uint, now time.Time) error {
	if agentID == 0 || userID == 0 {
		return fmt.Errorf("agent account is not linked")
	}
	row.Type = strings.ToUpper(strings.TrimSpace(row.Type))
	row.Reason = strings.TrimSpace(row.Reason)
	row.RequestedValue = strings.TrimSpace(row.RequestedValue)
	if !customerChangeTypes[row.Type] {
		return fmt.Errorf("invalid change request type")
	}
	if row.CustomerID == 0 || row.Reason == "" {
		return fmt.Errorf("customer and reason are required")
	}
	requested, err := parseCustomerChangeRequestedValue(row.Type, row.RequestedValue)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(requested)
	if err != nil {
		return fmt.Errorf("encode requested value: %w", err)
	}
	row.RequestedValue = string(encoded)
	customer, err := repositories.GetCustomerByID(row.CustomerID)
	if err != nil {
		return fmt.Errorf("customer not found")
	}
	if customer.AgentID == nil || *customer.AgentID != agentID {
		return fmt.Errorf("customer is outside this agent scope")
	}
	switch row.Type {
	case "PACKAGE":
		if err := ValidateAgentPackage(agentID, requested.PackageID); err != nil {
			return err
		}
	case "LINE_SHIFT":
		allowed, permissionErr := repositories.AgentHasRouter(agentID, requested.RouterID)
		if permissionErr != nil {
			return permissionErr
		}
		if !allowed {
			return fmt.Errorf("router is not assigned to this agent")
		}
		if err := ValidateSubscriptionRouter(requested.RouterID); err != nil {
			return err
		}
	}
	pending, err := repositories.PendingCustomerChangeRequestExists(row.CustomerID, row.Type)
	if err != nil {
		return err
	}
	if pending {
		return fmt.Errorf("a pending request of this type already exists")
	}
	row.RequestCode = fmt.Sprintf("CCR-%d-%d", now.UnixNano(), row.CustomerID)
	row.Status = "PENDING"
	row.AgentID = agentID
	row.RequestedByUserID = userID
	return repositories.CreateCustomerChangeRequest(row)
}

func ListCustomerChangeRequests(status string, agentID uint) ([]models.CustomerChangeRequest, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != "PENDING" && status != "COMPLETED" && status != "REJECTED" {
		return nil, fmt.Errorf("invalid status")
	}
	return repositories.ListCustomerChangeRequests(status, agentID)
}

type customerChangeRequestedValue struct {
	BillingDay int  `json:"billing_day"`
	PackageID  uint `json:"package_id"`
	RouterID   uint `json:"router_id"`
}

func parseCustomerChangeRequestedValue(requestType, raw string) (customerChangeRequestedValue, error) {
	var requested customerChangeRequestedValue
	if requestType == "CLOSE" {
		return requested, nil
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &requested); err != nil {
		return requested, fmt.Errorf("requested value is invalid")
	}
	switch requestType {
	case "BILLING_CYCLE":
		if requested.BillingDay < 1 || requested.BillingDay > 31 {
			return requested, fmt.Errorf("billing day must be between 1 and 31")
		}
	case "PACKAGE":
		if requested.PackageID == 0 {
			return requested, fmt.Errorf("package is required")
		}
	case "LINE_SHIFT":
		if requested.RouterID == 0 {
			return requested, fmt.Errorf("router is required")
		}
	default:
		return requested, fmt.Errorf("invalid change request type")
	}
	return requested, nil
}

type CustomerChangeRequestOption struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type CustomerChangeRequestOptions struct {
	Packages []CustomerChangeRequestOption `json:"packages"`
	Routers  []CustomerChangeRequestOption `json:"routers"`
}

func GetCustomerChangeRequestOptions(agentID uint) (*CustomerChangeRequestOptions, error) {
	if agentID == 0 {
		return nil, fmt.Errorf("agent account is not linked")
	}
	packages, err := ListAgentPackages(agentID)
	if err != nil {
		return nil, err
	}
	agent, err := GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found")
	}
	result := &CustomerChangeRequestOptions{
		Packages: make([]CustomerChangeRequestOption, 0, len(packages)),
		Routers:  make([]CustomerChangeRequestOption, 0, len(agent.AgentRouters)),
	}
	for _, item := range packages {
		result.Packages = append(result.Packages, CustomerChangeRequestOption{ID: item.ID, Code: item.PackageCode, Name: item.Name})
	}
	for _, assignment := range agent.AgentRouters {
		if !strings.EqualFold(assignment.Router.Status, "ACTIVE") {
			continue
		}
		result.Routers = append(result.Routers, CustomerChangeRequestOption{ID: assignment.Router.ID, Code: assignment.Router.Code, Name: assignment.Router.Name})
	}
	sort.Slice(result.Routers, func(i, j int) bool { return result.Routers[i].Name < result.Routers[j].Name })
	return result, nil
}

func ReviewCustomerChangeRequest(id, reviewerID uint, approve bool, reason string, now time.Time, keyMaterial string) (*models.CustomerChangeRequest, error) {
	row, err := repositories.GetCustomerChangeRequest(id)
	if err != nil {
		return nil, fmt.Errorf("request not found")
	}
	if row.Status != "PENDING" {
		return nil, fmt.Errorf("only pending requests can be reviewed")
	}
	if !approve && strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("rejection reason is required")
	}
	row.ReviewedByUserID = &reviewerID
	row.ReviewedAt = &now
	row.RejectionReason = strings.TrimSpace(reason)
	if !approve {
		result := database.DB.Model(&models.CustomerChangeRequest{}).
			Where("id = ? AND status = ?", row.ID, "PENDING").
			Updates(map[string]any{
				"status":              "REJECTED",
				"reviewed_by_user_id": reviewerID,
				"reviewed_at":         now,
				"rejection_reason":    row.RejectionReason,
			})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			return nil, fmt.Errorf("request was already reviewed")
		}
		row.Status = "REJECTED"
		return row, nil
	}

	requested, err := parseCustomerChangeRequestedValue(row.Type, row.RequestedValue)
	if err != nil {
		return nil, err
	}
	subscriptions, err := repositories.GetSubscriptionsByCustomer(row.CustomerID)
	if err != nil || len(subscriptions) == 0 {
		return nil, fmt.Errorf("customer subscription not found")
	}
	subscription := &subscriptions[0]
	oldRouterID := subscription.RouterID
	oldUsername := subscription.PPPoEUsername
	if strings.EqualFold(subscription.Status, "DISCONNECTED") {
		return nil, fmt.Errorf("subscription is already disconnected")
	}
	if row.Type == "PACKAGE" || row.Type == "LINE_SHIFT" {
		if _, err := GetSubscriptionPPPoEPassword(subscription, keyMaterial); err != nil {
			return nil, fmt.Errorf("PPPoE credential is required before approving this request: %w", err)
		}
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.CustomerChangeRequest{}).
			Where("id = ? AND status = ?", row.ID, "PENDING").
			Updates(map[string]any{
				"status":              "COMPLETED",
				"reviewed_by_user_id": reviewerID,
				"reviewed_at":         now,
				"rejection_reason":    "",
				"executed_at":         now,
				"execution_error":     "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("request was already reviewed")
		}

		updates, accountUpdates := map[string]any{}, map[string]any{}
		switch row.Type {
		case "BILLING_CYCLE":
			updates["billing_day"], accountUpdates["billing_day"] = requested.BillingDay, requested.BillingDay
		case "PACKAGE":
			if err := ValidateAgentPackage(row.AgentID, requested.PackageID); err != nil {
				return err
			}
			updates["package_id"], accountUpdates["package_id"] = requested.PackageID, requested.PackageID
		case "LINE_SHIFT":
			allowed, permissionErr := repositories.AgentHasRouter(row.AgentID, requested.RouterID)
			if permissionErr != nil {
				return permissionErr
			}
			if !allowed {
				return fmt.Errorf("router is not assigned to this agent")
			}
			if err := ValidateSubscriptionRouter(requested.RouterID); err != nil {
				return err
			}
			updates["router_id"], accountUpdates["router_id"] = requested.RouterID, requested.RouterID
		case "CLOSE":
			updates["status"], accountUpdates["status"] = "DISCONNECTED", "DISCONNECTED"
		default:
			return fmt.Errorf("unsupported request type")
		}
		if err := tx.Model(&models.Subscription{}).Where("id = ?", subscription.ID).Updates(updates).Error; err != nil {
			return err
		}
		if subscription.InternetAccountID != nil {
			if err := tx.Model(&models.CustomerInternetAccount{}).Where("id = ?", *subscription.InternetAccountID).Updates(accountUpdates).Error; err != nil {
				return err
			}
		}
		row.Status, row.ExecutedAt, row.ExecutionError = "COMPLETED", &now, ""
		return nil
	})
	if err != nil {
		return nil, err
	}
	subscription, err = repositories.GetSubscriptionByID(subscription.ID)
	if err != nil {
		return nil, err
	}
	if row.Type == "CLOSE" {
		_, err = ReconcileSubscriptionLifecycleWithMikroTikPostCommit(subscription, SubscriptionLifecycleDisconnect, keyMaterial)
	} else if row.Type == "LINE_SHIFT" {
		_, err = ReconcileSubscriptionPPPMigrationWithMikroTik(subscription.ID, oldRouterID, oldUsername, keyMaterial)
	} else if row.Type == "PACKAGE" {
		_, err = ReconcileSubscriptionPPPSecretWithMikroTik(subscription.ID, keyMaterial)
	}
	if err != nil {
		row.ExecutionError = err.Error()
		_ = repositories.SaveCustomerChangeRequest(row)
	}
	return row, nil
}
