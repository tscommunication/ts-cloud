package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func ExportData(c *gin.Context) {
	export, err := services.ExportData(c.Query("type"), c.DefaultQuery("format", "xlsx"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+export.Filename+`"`)
	c.Data(http.StatusOK, export.ContentType, export.Data)
}
