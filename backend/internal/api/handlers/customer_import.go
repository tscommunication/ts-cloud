package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func PreviewCustomerCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A CSV file up to 20 MB is required"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read CSV file"})
		return
	}
	defer opened.Close()
	preview, err := services.PreviewCustomerCSV(opened)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}

func ImportCustomerCSV(c *gin.Context) {
	routerID, err := strconv.ParseUint(c.PostForm("router_id"), 10, 64)
	if err != nil || routerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "router_id is required"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil || file.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A CSV file up to 20 MB is required"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read CSV file"})
		return
	}
	defer opened.Close()
	batch, err := services.ImportCustomerCSV(opened, file.Filename, uint(routerID))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, batch)
}
