package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func Me(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{
		"id":       c.GetUint("user_id"),
		"username": c.GetString("username"),
		"role":     c.GetString("role"),
		"agent_id": c.GetUint("agent_id"),
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

func ChangeMyPassword(c *gin.Context) {
	var req changePasswordRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password and a new password of at least 8 characters are required"})
		return
	}
	var user models.User
	if err := database.DB.First(&user, c.GetUint("user_id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword)) == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be different from the current password"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure new password"})
		return
	}
	if err := database.DB.Model(&user).Update("password", string(hash)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
