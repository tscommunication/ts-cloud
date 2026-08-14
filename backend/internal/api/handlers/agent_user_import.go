package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/services"
)

const maxAgentUserImportSize = 20 * 1024 * 1024

func PreviewAgentUserImport(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file.Size > maxAgentUserImportSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "An XLSX file up to 20 MB is required"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read import file"})
		return
	}
	defer opened.Close()
	preview, err := services.PreviewAgentUserWorkbook(opened, file.Filename)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}

func ImportAgentUsers(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file.Size > maxAgentUserImportSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "An XLSX file up to 20 MB is required"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read import file"})
		return
	}
	defer opened.Close()
	result, err := services.ImportAgentUserWorkbook(opened, file.Filename)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}
