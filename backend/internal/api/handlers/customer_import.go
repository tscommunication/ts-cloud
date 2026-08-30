package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func PreviewCustomerCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A CSV or XLSX file up to 20 MB is required"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read import file"})
		return
	}
	defer opened.Close()
	preview, err := services.PreviewCustomerFile(opened, file.Filename)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}

func ImportCustomerCSV(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		routerID, err := strconv.ParseUint(c.PostForm("router_id"), 10, 64)
		if err != nil || routerID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "router_id is required"})
			return
		}
		file, err := c.FormFile("file")
		if err != nil || file.Size > 20*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "A CSV or XLSX file up to 20 MB is required"})
			return
		}
		opened, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read import file"})
			return
		}
		defer opened.Close()
		batch, err := services.ImportCustomerFileWithCredentialKey(opened, file.Filename, uint(routerID), cfg.CredentialKey)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, batch)
	}
}
