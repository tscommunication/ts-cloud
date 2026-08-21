package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetUsers(c *gin.Context) {

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	search := c.DefaultQuery("search", "")
	sort := c.DefaultQuery("sort", "id")
	order := c.DefaultQuery("order", "desc")

	users, total, err := services.GetUsers(
		page,
		limit,
		search,
		sort,
		order,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":  page,
		"limit": limit,
		"total": total,
		"count": len(users),
		"users": dto.ToUserResponses(users),
	})
}

func GetUser(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}
	actorID := c.GetUint("user_id")
	actorRole := c.GetString("role")
	if actorRole != "superadmin" && actorRole != "admin" && actorID != uint(id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	user, err := services.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToUserResponse(*user))
}

func CreateUser(c *gin.Context) {

	var req dto.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	if req.Role != "" && req.Role != "superadmin" && req.Role != "admin" && req.Role != "user" && req.Role != "agent" && req.Role != "customer" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Role must be superadmin, admin, agent, customer or user",
		})
		return
	}
	if req.Role == "customer" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Customer accounts are managed from the Customer module",
		})
		return
	}

	if _, err := services.GetUserByUsername(req.Username); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Username already exists",
		})
		return
	}

	if _, err := services.GetUserByEmail(req.Email); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Email already exists",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to hash password",
		})
		return
	}

	role := req.Role
	if role == "" {
		role = "user"
	}
	switch role {
	case "agent":
		if req.AgentID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Agent is required for agent role"})
			return
		}
		if _, err := services.GetAgent(*req.AgentID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Agent not found"})
			return
		}
		req.CustomerID = nil

	default:
		req.AgentID = nil
		req.CustomerID = nil
	}

	user := models.User{
		Name:       req.Name,
		Username:   req.Username,
		Email:      req.Email,
		Password:   string(hashedPassword),
		Role:       role,
		Active:     true,
		AgentID:    req.AgentID,
		CustomerID: req.CustomerID,
	}

	if err := services.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	c.JSON(http.StatusCreated, dto.ToUserResponse(user))
}

func UpdateUser(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var req dto.UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	user, err := services.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	if user.Role == "customer" || req.Role == "customer" || req.CustomerID != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Customer accounts are managed from the Customer module",
		})
		return
	}

	actorID := c.GetUint("user_id")
	actorRole := c.GetString("role")
	if actorRole != "superadmin" && actorRole != "admin" && actorID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	if actorRole == "admin" && user.Role == "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	if req.Role != "" && req.Role != "superadmin" && req.Role != "admin" && req.Role != "agent" && req.Role != "customer" && req.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role must be superadmin, admin, agent, customer or user"})
		return
	}
	if actorRole != "superadmin" && (req.Role != "" || req.Active != nil || req.AgentID != nil || req.CustomerID != nil) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only superadmin can change roles or account status"})
		return
	}
	if actorID == user.ID && req.Active != nil && !*req.Active {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot disable your own account"})
		return
	}
	if actorID == user.ID && req.Role != "" && req.Role != user.Role {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot change your own role"})
		return
	}
	removesActiveSuperadmin := user.Role == "superadmin" && user.Active && ((req.Role != "" && req.Role != "superadmin") || (req.Active != nil && !*req.Active))
	if removesActiveSuperadmin {
		count, countErr := services.CountActiveSuperadmins()
		if countErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify superadmin safety"})
			return
		}
		if count <= 1 {
			c.JSON(http.StatusConflict, gin.H{"error": "The last active superadmin cannot be disabled or demoted"})
			return
		}
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Username != "" {
		user.Username = req.Username
	}

	if req.Email != "" {
		user.Email = req.Email
	}

	if req.Role != "" {
		user.Role = req.Role
	}
	if actorRole == "superadmin" &&
		(req.Role != "" || req.AgentID != nil || req.CustomerID != nil) {

		switch user.Role {
		case "agent":
			user.CustomerID = nil

			if req.AgentID != nil {
				if _, err := services.GetAgent(*req.AgentID); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Agent not found"})
					return
				}
				user.AgentID = req.AgentID
			}

			if user.AgentID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Agent is required for agent role"})
				return
			}

		default:
			user.AgentID = nil
			user.CustomerID = nil
		}
	}

	if req.Active != nil {
		user.Active = *req.Active
	}

	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(req.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to hash password",
			})
			return
		}

		user.Password = string(hashedPassword)
	}

	if err := services.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToUserResponse(*user))
}
func DeleteUser(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	if uint(id) == c.GetUint("user_id") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot delete your own account"})
		return
	}

	user, err := services.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if user.Role == "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Superadmin accounts cannot be deleted through the API"})
		return
	}
	if user.Role == "customer" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer accounts are managed from the Customer module",
		})
		return
	}

	if err := services.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}
