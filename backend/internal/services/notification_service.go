package services

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type UserNotification struct {
	models.Notification
	Read bool `json:"read"`
}

func createCustomerNotification(tx *gorm.DB, customer *models.Customer) error {
	item := models.Notification{
		Type: "CUSTOMER_CREATED", Severity: "INFO", Title: "Customer created",
		Message:    fmt.Sprintf("%s (%s) was created", customer.FullName, customer.CustomerCode),
		EntityType: "CUSTOMER", EntityID: customer.ID, TargetPath: fmt.Sprintf("/customers?customer=%d", customer.ID),
		DedupKey: fmt.Sprintf("customer-created:%d", customer.ID), Active: true,
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedup_key"}}, DoNothing: true}).Create(&item).Error
}

func createCustomerChangeRequestNotification(
	tx *gorm.DB,
	request *models.CustomerChangeRequest,
	customer *models.Customer,
) error {
	item := models.Notification{
		Type:       "CUSTOMER_CHANGE_REQUEST",
		Severity:   "WARNING",
		Title:      "Customer change request pending",
		Message:    fmt.Sprintf("%s (%s): %s request awaits review", customer.FullName, customer.CustomerCode, strings.ReplaceAll(strings.ToLower(request.Type), "_", " ")),
		EntityType: "CUSTOMER_CHANGE_REQUEST",
		EntityID:   request.ID,
		TargetPath: "/customer-change-requests",
		DedupKey:   fmt.Sprintf("customer-change-request:%d", request.ID),
		Active:     true,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "dedup_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"severity", "title", "message", "active", "updated_at"}),
	}).Create(&item).Error
}

func resolveCustomerChangeRequestNotification(tx *gorm.DB, requestID uint) error {
	return tx.Model(&models.Notification{}).
		Where("dedup_key = ?", fmt.Sprintf("customer-change-request:%d", requestID)).
		Update("active", false).Error
}

func createCustomerChangeRequestReviewNotification(
	tx *gorm.DB,
	request *models.CustomerChangeRequest,
	customer *models.Customer,
	approved bool,
) error {
	title, severity, decision := "Customer change request rejected", "WARNING", "rejected"
	if approved {
		title, severity, decision = "Customer change request approved", "INFO", "approved"
	}
	message := fmt.Sprintf("%s (%s): your %s request was %s", customer.FullName, customer.CustomerCode, strings.ReplaceAll(strings.ToLower(request.Type), "_", " "), decision)
	if !approved && request.RejectionReason != "" {
		message = fmt.Sprintf("%s. Reason: %s", message, request.RejectionReason)
	}
	recipientUserID := request.RequestedByUserID
	item := models.Notification{
		Type:            "CUSTOMER_CHANGE_REQUEST_REVIEWED",
		Severity:        severity,
		Title:           title,
		Message:         message,
		EntityType:      "CUSTOMER_CHANGE_REQUEST",
		EntityID:        request.ID,
		RecipientUserID: &recipientUserID,
		TargetPath:      "/customer-change-requests",
		DedupKey:        fmt.Sprintf("customer-change-request:%d:review", request.ID),
		Active:          true,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "dedup_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"severity", "title", "message", "recipient_user_id", "active", "updated_at"}),
	}).Create(&item).Error
}

func SyncNetworkAlertNotification(alert *models.NetworkRouterAlert, router *models.NetworkRouter) error {
	item := models.Notification{
		Type: "NETWORK_ALERT", Severity: alert.Severity, Title: "Active network alert", Message: alert.Message,
		EntityType: "NETWORK_ROUTER_ALERT", EntityID: alert.ID, TargetPath: "/network/routers",
		DedupKey: fmt.Sprintf("network-alert:%d", alert.ID), Active: alert.Status == "ACTIVE",
	}
	if router != nil && router.Code != "" {
		item.Title = fmt.Sprintf("Network alert · %s", router.Code)
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "dedup_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"severity", "title", "message", "active", "updated_at"}),
	}).Create(&item).Error
}

// SyncOLTOfflineNotification keeps one active notification per OLT and
// resolves it when polling shows the OLT has recovered.
func SyncOLTOfflineNotification(device *models.NetworkDevice, offline bool, reason string) error {
	if device == nil || device.ID == 0 || !strings.EqualFold(device.DeviceType, "OLT") {
		return nil
	}
	message := fmt.Sprintf("OLT %s is unreachable", device.Code)
	if reason = strings.TrimSpace(reason); reason != "" {
		message += ": " + reason
	}
	item := models.Notification{
		Type: "OLT_OFFLINE", Severity: "CRITICAL", Title: fmt.Sprintf("OLT offline · %s", device.Code), Message: message,
		EntityType: "NETWORK_DEVICE", EntityID: device.ID, TargetPath: "/network/devices?type=OLT",
		DedupKey: fmt.Sprintf("olt-offline:%d", device.ID), Active: offline,
	}
	if err := database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "dedup_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"severity", "title", "message", "active", "updated_at"}),
	}).Create(&item).Error; err != nil {
		return err
	}
	if !offline {
		return database.DB.Model(&models.Notification{}).
			Where("dedup_key LIKE ?", fmt.Sprintf("olt-offline:%d:agent:%%", device.ID)).
			Update("active", false).Error
	}

	var recipients []uint
	if err := database.DB.Model(&models.User{}).
		Joins("JOIN agent_network_devices ON agent_network_devices.agent_id = users.agent_id").
		Where("agent_network_devices.network_device_id = ? AND users.role = ? AND users.active = ?", device.ID, "agent", true).
		Distinct("users.id").Pluck("users.id", &recipients).Error; err != nil {
		return err
	}
	for _, userID := range recipients {
		recipientID := userID
		agentItem := models.Notification{
			Type:            item.Type,
			Severity:        item.Severity,
			Title:           item.Title,
			Message:         item.Message,
			EntityType:      item.EntityType,
			EntityID:        item.EntityID,
			RecipientUserID: &recipientID,
			TargetPath:      item.TargetPath,
			DedupKey:        fmt.Sprintf("olt-offline:%d:agent:%d", device.ID, userID),
			Active:          true,
		}
		if err := database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedup_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"severity", "title", "message", "recipient_user_id", "active", "updated_at"}),
		}).Create(&agentItem).Error; err != nil {
			return err
		}
	}
	return nil
}

func notificationVisibilityQuery(query *gorm.DB, userID uint, role string) *gorm.DB {
	if role == "agent" {
		return query.Where("n.recipient_user_id = ?", userID)
	}
	return query.Where("n.recipient_user_id IS NULL OR n.recipient_user_id = ?", userID)
}

func ListUserNotifications(userID uint, role string, limit int) ([]UserNotification, int64, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}
	var rows []UserNotification
	err := notificationVisibilityQuery(database.DB.Table("notifications n"), userID, role).
		Select("n.*, CASE WHEN nr.id IS NULL THEN false ELSE true END AS read").
		Joins("LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?", userID).
		Where("n.active = ?", true).Order("n.created_at DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	var unread int64
	err = notificationVisibilityQuery(database.DB.Table("notifications n"), userID, role).
		Joins("LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?", userID).
		Where("n.active = ? AND nr.id IS NULL", true).Count(&unread).Error
	return rows, unread, err
}

func MarkNotificationRead(userID, notificationID uint, role string) error {
	var count int64
	query := database.DB.Table("notifications n").Where("n.id = ? AND n.active = ?", notificationID, true)
	if err := notificationVisibilityQuery(query, userID, role).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	item := models.NotificationRead{NotificationID: notificationID, UserID: userID, ReadAt: time.Now()}
	return database.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "notification_id"}, {Name: "user_id"}}, DoNothing: true}).Create(&item).Error
}

func MarkAllNotificationsRead(userID uint, role string) error {
	condition := "active = true AND (recipient_user_id IS NULL OR recipient_user_id = ?)"
	if role == "agent" {
		condition = "active = true AND recipient_user_id = ?"
	}
	return database.DB.Exec(`INSERT INTO notification_reads (notification_id, user_id, read_at)
		SELECT id, ?, ? FROM notifications WHERE `+condition+`
		ON CONFLICT (notification_id, user_id) DO NOTHING`, userID, time.Now(), userID).Error
}
