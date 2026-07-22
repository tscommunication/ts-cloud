package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

// GetPackages godoc
//
//	@Summary		Get Packages
//	@Description	Get all ISP packages
//	@Tags			Packages
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {object} map[string]interface{}
//	@Router			/api/v1/packages [get]
func GetPackages(c *gin.Context) {

	packages, err := services.GetPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch packages",
		})
		return
	}

	response := make([]dto.PackageResponse, 0)

	for _, pkg := range packages {
		response = append(response, dto.ToPackageResponse(pkg))
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(response),
		"packages": response,
	})
}

// GetPackage godoc
//
//	@Summary		Get Package
//	@Description	Get package by ID
//	@Tags			Packages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	int	true	"Package ID"
//	@Success		200 {object} dto.PackageResponse
//	@Failure		404 {object} map[string]interface{}
//	@Router			/api/v1/packages/{id} [get]
func GetPackage(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid package ID",
		})
		return
	}

	pkg, err := services.GetPackageByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Package not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToPackageResponse(*pkg))
}

// CreatePackage godoc
//
//	@Summary		Create Package
//	@Description	Create a new ISP package
//	@Tags			Packages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body	dto.CreatePackageRequest	true	"Package"
//	@Success		201 {object} dto.PackageResponse
//	@Failure		400 {object} map[string]interface{}
//	@Router			/api/v1/packages [post]
func CreatePackage(c *gin.Context) {

	var req dto.CreatePackageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	packageCode := "PKG-000001"

	lastPackage, err := services.GetLastPackage()
	if err == nil {
		packageCode = fmt.Sprintf("PKG-%06d", lastPackage.ID+1)
	}

	pkg := models.Package{
		PackageCode:     packageCode,
		Name:            req.Name,
		Price:           req.Price,
		DownloadSpeed:   req.DownloadSpeed,
		UploadSpeed:     req.UploadSpeed,
		BurstDownload:   req.BurstDownload,
		BurstUpload:     req.BurstUpload,
		ValidityDays:    req.ValidityDays,
		MikroTikProfile: req.MikroTikProfile,
		RadiusProfile:   req.RadiusProfile,
		Description:     req.Description,
		Status:          "ACTIVE",
	}
	if err := services.CreatePackage(&pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create package",
		})
		return
	}

	c.JSON(http.StatusCreated, dto.ToPackageResponse(pkg))
}

// UpdatePackage godoc
//
//	@Summary		Update Package
//	@Description	Update package information
//	@Tags			Packages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	int							true	"Package ID"
//	@Param			request	body	dto.CreatePackageRequest	true	"Package"
//	@Success		200		{object}	dto.PackageResponse
//	@Failure		404		{object}	map[string]interface{}
//	@Router			/api/v1/packages/{id} [put]
func UpdatePackage(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid package ID",
		})
		return
	}

	pkg, err := services.GetPackageByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Package not found",
		})
		return
	}

	var req dto.CreatePackageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	pkg.Name = req.Name
	pkg.Price = req.Price
	pkg.DownloadSpeed = req.DownloadSpeed
	pkg.UploadSpeed = req.UploadSpeed
	pkg.BurstDownload = req.BurstDownload
	pkg.BurstUpload = req.BurstUpload
	pkg.ValidityDays = req.ValidityDays
	pkg.MikroTikProfile = req.MikroTikProfile
	pkg.RadiusProfile = req.RadiusProfile
	pkg.Description = req.Description

	if err := services.UpdatePackage(pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update package",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToPackageResponse(*pkg))
}

// DeletePackage godoc
//
//	@Summary		Delete Package
//	@Description	Delete package by ID
//	@Tags			Packages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	int	true	"Package ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Router			/api/v1/packages/{id} [delete]
func DeletePackage(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid package ID",
		})
		return
	}

	if _, err := services.GetPackageByID(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Package not found",
		})
		return
	}

	if err := services.DeletePackage(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete package",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Package deleted successfully",
	})
}
