package services

import (
	"fmt"
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

func ListUserNotifications(userID uint, limit int) ([]UserNotification, int64, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}
	var rows []UserNotification
	err := database.DB.Table("notifications n").
		Select("n.*, CASE WHEN nr.id IS NULL THEN false ELSE true END AS read").
		Joins("LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?", userID).
		Where("n.active = ?", true).Order("n.created_at DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	var unread int64
	err = database.DB.Table("notifications n").
		Joins("LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?", userID).
		Where("n.active = ? AND nr.id IS NULL", true).Count(&unread).Error
	return rows, unread, err
}

func MarkNotificationRead(userID, notificationID uint) error {
	var count int64
	if err := database.DB.Model(&models.Notification{}).Where("id = ? AND active = ?", notificationID, true).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	item := models.NotificationRead{NotificationID: notificationID, UserID: userID, ReadAt: time.Now()}
	return database.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "notification_id"}, {Name: "user_id"}}, DoNothing: true}).Create(&item).Error
}

func MarkAllNotificationsRead(userID uint) error {
	return database.DB.Exec(`INSERT INTO notification_reads (notification_id, user_id, read_at)
		SELECT id, ?, ? FROM notifications WHERE active = true
		ON CONFLICT (notification_id, user_id) DO NOTHING`, userID, time.Now()).Error
}
