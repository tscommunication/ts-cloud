package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func parseLocationParentID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid location id",
		})
		return 0, false
	}

	return uint(value), true
}

func GetDivisions(c *gin.Context) {
	rows, err := services.ListDivisions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rows)
}

func GetDistrictsByDivision(c *gin.Context) {
	divisionID, ok := parseLocationParentID(c)
	if !ok {
		return
	}

	rows, err := services.ListDistrictsByDivision(divisionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rows)
}

func GetUpazilasByDistrict(c *gin.Context) {
	districtID, ok := parseLocationParentID(c)
	if !ok {
		return
	}

	rows, err := services.ListUpazilasByDistrict(districtID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rows)
}

func GetPostOfficesByUpazila(c *gin.Context) {
	upazilaID, ok := parseLocationParentID(c)
	if !ok {
		return
	}

	rows, err := services.ListPostOfficesByUpazila(upazilaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rows)
}
