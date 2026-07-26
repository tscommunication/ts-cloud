package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

// CreateFTPServer godoc
//
//	@Summary		Create FTP Server
//	@Description	Create a new FTP server
//	@Tags			FTP Server
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateFTPServerRequest	true	"FTP Server"
//	@Success		201		{object}	dto.APIResponse
//	@Failure		400		{object}	dto.APIResponse
//	@Failure		500		{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-servers [post]
func CreateFTPServer(c *gin.Context) {

	var req dto.CreateFTPServerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error(err.Error()),
		)
		return
	}

	server := models.FTPServer{
		Name:             req.Name,
		Driver:           req.Driver,
		Host:             req.Host,
		Port:             req.Port,
		Username:         req.Username,
		Password:         req.Password,
		RootPath:         req.RootPath,
		PassivePortStart: req.PassivePortStart,
		PassivePortEnd:   req.PassivePortEnd,
		MaxConnections:   req.MaxConnections,
		Status:           req.Status,
		Description:      req.Description,
	}

	if err := services.CreateFTPServer(&server); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	c.JSON(
		http.StatusCreated,
		dto.Success(
			dto.ToFTPServerResponse(server),
		),
	)
}

// GetFTPServers godoc
//
//	@Summary		List FTP Servers
//	@Description	Get all FTP servers
//	@Tags			FTP Server
//	@Produce		json
//	@Success		200	{object}	dto.APIResponse
//	@Failure		500	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-servers [get]
func GetFTPServers(c *gin.Context) {

	servers, err := services.GetFTPServers()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	response := make([]dto.FTPServerResponse, 0)

	for _, server := range servers {
		response = append(
			response,
			dto.ToFTPServerResponse(server),
		)
	}

	c.JSON(
		http.StatusOK,
		dto.Success(response),
	)
}

// GetFTPServerByID godoc
//
//	@Summary		Get FTP Server
//	@Description	Get FTP server by ID
//	@Tags			FTP Server
//	@Produce		json
//	@Param			id	path		int	true	"FTP Server ID"
//	@Success		200	{object}	dto.APIResponse
//	@Failure		400	{object}	dto.APIResponse
//	@Failure		404	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-servers/{id} [get]
func GetFTPServerByID(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error("invalid id"),
		)
		return
	}

	server, err := services.GetFTPServerByID(uint(id))
	if err != nil {
		c.JSON(
			http.StatusNotFound,
			dto.Error(err.Error()),
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.Success(
			dto.ToFTPServerResponse(*server),
		),
	)
}

// UpdateFTPServer godoc
//
//	@Summary		Update FTP Server
//	@Description	Update FTP server information
//	@Tags			FTP Server
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"FTP Server ID"
//	@Param			request	body		dto.CreateFTPServerRequest	true	"FTP Server"
//	@Success		200		{object}	dto.APIResponse
//	@Failure		400		{object}	dto.APIResponse
//	@Failure		404		{object}	dto.APIResponse
//	@Failure		500		{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-servers/{id} [put]
func UpdateFTPServer(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error("invalid id"),
		)
		return
	}

	server, err := services.GetFTPServerByID(uint(id))
	if err != nil {
		c.JSON(
			http.StatusNotFound,
			dto.Error(err.Error()),
		)
		return
	}

	var req dto.CreateFTPServerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error(err.Error()),
		)
		return
	}

	server.Name = req.Name
	server.Driver = req.Driver
	server.Host = req.Host
	server.Port = req.Port
	server.Username = req.Username
	server.Password = req.Password
	server.RootPath = req.RootPath
	server.PassivePortStart = req.PassivePortStart
	server.PassivePortEnd = req.PassivePortEnd
	server.MaxConnections = req.MaxConnections
	server.Status = req.Status
	server.Description = req.Description

	if err := services.UpdateFTPServer(server); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.Success(
			dto.ToFTPServerResponse(*server),
		),
	)
}

// DeleteFTPServer godoc
//
//	@Summary		Delete FTP Server
//	@Description	Delete FTP server by ID
//	@Tags			FTP Server
//	@Produce		json
//	@Param			id	path		int	true	"FTP Server ID"
//	@Success		200	{object}	dto.APIResponse
//	@Failure		400	{object}	dto.APIResponse
//	@Failure		500	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-servers/{id} [delete]
func DeleteFTPServer(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error("invalid id"),
		)
		return
	}

	if err := services.DeleteFTPServer(uint(id)); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.SuccessMessage("FTP Server deleted successfully"),
	)
}
