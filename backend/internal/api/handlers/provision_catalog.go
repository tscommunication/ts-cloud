package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/services"
)

type provisionPackageResponse struct {
	ID            uint    `json:"id"`
	PackageCode   string  `json:"package_code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	DownloadSpeed int     `json:"download_speed"`
	UploadSpeed   int     `json:"upload_speed"`
	Status        string  `json:"status"`
}

type provisionRouterResponse struct {
	ID      uint   `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	POPID   *uint  `json:"pop_id,omitempty"`
	POPName string `json:"pop_name"`
	Status  string `json:"status"`
}

func GetProvisionCatalogPackages(c *gin.Context) {
	rows, err := services.ListProvisionCatalogPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load package catalog",
		})
		return
	}

	response := make([]provisionPackageResponse, 0, len(rows))

	for _, row := range rows {
		response = append(response, provisionPackageResponse{
			ID:            row.ID,
			PackageCode:   row.PackageCode,
			Name:          row.Name,
			Price:         row.Price,
			DownloadSpeed: row.DownloadSpeed,
			UploadSpeed:   row.UploadSpeed,
			Status:        row.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(response),
		"packages": response,
	})
}

func GetProvisionCatalogRouters(c *gin.Context) {
	rows, err := services.ListProvisionCatalogRouters(
		c.GetString("role"),
		c.GetUint("agent_id"),
	)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := make([]provisionRouterResponse, 0, len(rows))

	for _, row := range rows {
		item := provisionRouterResponse{
			ID:     row.ID,
			Code:   row.Code,
			Name:   row.Name,
			POPID:  row.POPID,
			Status: row.Status,
		}

		if row.POP != nil {
			item.POPName = row.POP.Name
		}

		response = append(response, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(response),
		"routers": response,
	})
}
