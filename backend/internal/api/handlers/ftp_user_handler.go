package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

// CreateFTPUser godoc
//
//	@Summary		Create FTP User
//	@Description	Create a new FTP user
//	@Tags			FTP User
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateFTPUserRequest	true	"FTP User"
//	@Success		201		{object}	dto.APIResponse
//	@Failure		400		{object}	dto.APIResponse
//	@Failure		500		{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-users [post]
func CreateFTPUser(c *gin.Context) {

	var req dto.CreateFTPUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error(err.Error()),
		)
		return
	}

	user := models.FTPUser{
		SubscriptionID:    req.SubscriptionID,
		FTPServerID:       req.FTPServerID,
		Username:          req.Username,
		Password:          req.Password,
		HomeDirectory:     req.HomeDirectory,
		StorageQuotaGB:    req.StorageQuotaGB,
		UploadLimitMbps:   req.UploadLimitMbps,
		DownloadLimitMbps: req.DownloadLimitMbps,
		Status:            req.Status,
		Remarks:           req.Remarks,
	}

	if err := services.CreateFTPUser(&user); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	c.JSON(
		http.StatusCreated,
		dto.Success(
			dto.ToFTPUserResponse(user),
		),
	)
}

// GetFTPUsers godoc
//
//	@Summary		List FTP Users
//	@Description	Get all FTP users
//	@Tags			FTP User
//	@Produce		json
//	@Success		200	{object}	dto.APIResponse
//	@Failure		500	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-users [get]
func GetFTPUsers(c *gin.Context) {

	users, err := services.GetFTPUsers()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	response := make([]dto.FTPUserResponse, 0)

	for _, user := range users {
		response = append(
			response,
			dto.ToFTPUserResponse(user),
		)
	}

	c.JSON(
		http.StatusOK,
		dto.Success(response),
	)
}

// GetFTPUserByID godoc
//
//	@Summary		Get FTP User
//	@Description	Get FTP user by ID
//	@Tags			FTP User
//	@Produce		json
//	@Param			id	path		int	true	"FTP User ID"
//	@Success		200	{object}	dto.APIResponse
//	@Failure		400	{object}	dto.APIResponse
//	@Failure		404	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-users/{id} [get]
func GetFTPUserByID(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error("invalid id"),
		)
		return
	}

	user, err := services.GetFTPUserByID(uint(id))
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
			dto.ToFTPUserResponse(*user),
		),
	)
}

// UpdateFTPUser godoc
//
//	@Summary		Update FTP User
//	@Description	Update FTP user information
//	@Tags			FTP User
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"FTP User ID"
//	@Param			request	body		dto.CreateFTPUserRequest	true	"FTP User"
//	@Success		200		{object}	dto.APIResponse
//	@Failure		400		{object}	dto.APIResponse
//	@Failure		404		{object}	dto.APIResponse
//	@Failure		500		{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-users/{id} [put]
func UpdateFTPUser(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error("invalid id"),
		)
		return
	}

	user, err := services.GetFTPUserByID(uint(id))
	if err != nil {
		c.JSON(
			http.StatusNotFound,
			dto.Error(err.Error()),
		)
		return
	}

	var req dto.CreateFTPUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error(err.Error()),
		)
		return
	}

	user.SubscriptionID = req.SubscriptionID
	user.FTPServerID = req.FTPServerID
	user.Username = req.Username
	user.Password = req.Password
	user.HomeDirectory = req.HomeDirectory
	user.StorageQuotaGB = req.StorageQuotaGB
	user.UploadLimitMbps = req.UploadLimitMbps
	user.DownloadLimitMbps = req.DownloadLimitMbps
	user.Status = req.Status
	user.Remarks = req.Remarks

	if err := services.UpdateFTPUser(user); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.Success(
			dto.ToFTPUserResponse(*user),
		),
	)
}

// DeleteFTPUser godoc
//
//	@Summary		Delete FTP User
//	@Description	Delete FTP user by ID
//	@Tags			FTP User
//	@Produce		json
//	@Param			id	path		int	true	"FTP User ID"
//	@Success		200	{object}	dto.APIResponse
//	@Failure		400	{object}	dto.APIResponse
//	@Failure		500	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-users/{id} [delete]
func DeleteFTPUser(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			dto.Error("invalid id"),
		)
		return
	}

	if err := services.DeleteFTPUser(uint(id)); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(err.Error()),
		)
		return
	}

	c.JSON(
		http.StatusOK,
		dto.SuccessMessage("FTP User deleted successfully"),
	)
}

// SuspendFTPUser godoc
//
//	@Summary		Suspend FTP User
//	@Description	Lock Linux FTP account
//	@Tags			FTP User
//	@Produce		json
//	@Param			id	path		int	true	"FTP User ID"
//	@Success		200	{object}	dto.APIResponse
//	@Failure		400	{object}	dto.APIResponse
//	@Failure		500	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-users/{id}/suspend [post]
func SuspendFTPUser(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("invalid id"))
		return
	}

	if err := services.SuspendFTPUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err.Error()))
		return
	}

	c.JSON(
		http.StatusOK,
		dto.SuccessMessage("FTP User suspended successfully"),
	)
}

// EnableFTPUser godoc
//
//	@Summary		Enable FTP User
//	@Description	Unlock Linux FTP account
//	@Tags			FTP User
//	@Produce		json
//	@Param			id	path		int	true	"FTP User ID"
//	@Success		200	{object}	dto.APIResponse
//	@Failure		400	{object}	dto.APIResponse
//	@Failure		500	{object}	dto.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/ftp-users/{id}/enable [post]
func EnableFTPUser(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("invalid id"))
		return
	}

	if err := services.EnableFTPUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err.Error()))
		return
	}

	c.JSON(
		http.StatusOK,
		dto.SuccessMessage("FTP User enabled successfully"),
	)
}
