package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/tscommunication/ts-cloud/internal/auth"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"tsadmin"`
	Password string `json:"password" binding:"required" example:"admin123"`
}

// Login godoc
//
//	@Summary		User Login
//	@Description	Login using username and password
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"Login Request"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Router			/api/v1/auth/login [post]
func Login(c *gin.Context) {

	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	user, err := services.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	); err != nil {

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	token, err := auth.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Login successful",
		"access_token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
			"agent_id": user.AgentID,
		},
	})
}
