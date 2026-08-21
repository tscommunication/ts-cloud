package models

import "time"

type Notification struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Type       string    `gorm:"size:40;not null;index" json:"type"`
	Severity   string    `gorm:"size:20;not null;index" json:"severity"`
	Title      string    `gorm:"size:160;not null" json:"title"`
	Message    string    `gorm:"size:500;not null" json:"message"`
	EntityType string    `gorm:"size:40;not null;index" json:"entity_type"`
	EntityID   uint      `gorm:"not null;index" json:"entity_id"`
	TargetPath string    `gorm:"size:255;not null" json:"target_path"`
	DedupKey   string    `gorm:"size:160;not null;uniqueIndex" json:"-"`
	Active     bool      `gorm:"not null;index" json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type NotificationRead struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	NotificationID uint          `gorm:"not null;uniqueIndex:idx_notification_user_read;index" json:"notification_id"`
	Notification   *Notification `gorm:"foreignKey:NotificationID;constraint:OnDelete:CASCADE" json:"-"`
	UserID         uint          `gorm:"not null;uniqueIndex:idx_notification_user_read;index" json:"user_id"`
	User           *User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	ReadAt         time.Time     `gorm:"not null" json:"read_at"`
}
