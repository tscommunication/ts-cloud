package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetNotifications(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	items, unread, err := services.ListUserNotifications(c.GetUint("user_id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": items, "unread_count": unread})
}

func MarkNotificationRead(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}
	err = services.MarkNotificationRead(c.GetUint("user_id"), uint(id))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification read"})
		return
	}
	c.Status(http.StatusNoContent)
}

func MarkAllNotificationsRead(c *gin.Context) {
	if err := services.MarkAllNotificationsRead(c.GetUint("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications read"})
		return
	}
	c.Status(http.StatusNoContent)
}
