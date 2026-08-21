package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

var managedCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,29}$`)

type updateManagedCodeRequest struct {
	Code   string `json:"code" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

func UpdateManagedCode(c *gin.Context) {
	entityType := strings.ToUpper(strings.TrimSpace(c.Param("entity")))
	idValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || idValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid record ID"})
		return
	}
	var req updateManagedCodeRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and reason are required"})
		return
	}
	newCode := strings.ToUpper(strings.TrimSpace(req.Code))
	reason := strings.TrimSpace(req.Reason)
	if !managedCodePattern.MatchString(newCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code must be 2-30 characters using A-Z, 0-9, hyphen or underscore"})
		return
	}
	if reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason is required for audit"})
		return
	}

	entityID := uint(idValue)
	var oldCode string
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var model any
		var codeColumn string
		switch entityType {
		case "AGENT":
			row := &models.Agent{}
			if err := tx.Unscoped().First(row, entityID).Error; err != nil {
				return err
			}
			oldCode, model, codeColumn = row.Code, row, "code"
		case "POP":
			row := &models.POP{}
			if err := tx.Unscoped().First(row, entityID).Error; err != nil {
				return err
			}
			oldCode, model, codeColumn = row.Code, row, "code"
		case "PACKAGE":
			row := &models.Package{}
			if err := tx.Unscoped().First(row, entityID).Error; err != nil {
				return err
			}
			oldCode, model, codeColumn = row.PackageCode, row, "package_code"
		default:
			return fmt.Errorf("unsupported entity type")
		}
		if strings.EqualFold(oldCode, newCode) {
			return fmt.Errorf("new code is unchanged")
		}
		if err := tx.Model(model).Update(codeColumn, newCode).Error; err != nil {
			return fmt.Errorf("code already exists or cannot be updated")
		}
		return tx.Create(&models.CodeChangeAudit{
			EntityType: entityType, EntityID: entityID, OldCode: oldCode,
			NewCode: newCode, ChangedBy: c.GetUint("user_id"), Reason: reason,
			ChangedAt: time.Now(),
		}).Error
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"entity_type": entityType, "entity_id": entityID,
		"old_code": oldCode, "new_code": newCode,
	})
}
