package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type networkRouterRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	POPID       *uint  `json:"pop_id"`
	Host        string `json:"host" binding:"required"`
	APIPort     int    `json:"api_port" binding:"required"`
	APIUsername string `json:"api_username" binding:"required"`
	UseTLS      bool   `json:"use_tls"`
	Status      string `json:"status" binding:"required"`
	Remarks     string `json:"remarks"`
}

type networkRouterResponse struct {
	ID                    uint       `json:"id"`
	Code                  string     `json:"code"`
	Name                  string     `json:"name"`
	POPID                 *uint      `json:"pop_id"`
	POPName               string     `json:"pop_name"`
	Host                  string     `json:"host"`
	APIPort               int        `json:"api_port"`
	APIUsername           string     `json:"api_username"`
	UseTLS                bool       `json:"use_tls"`
	Status                string     `json:"status"`
	Remarks               string     `json:"remarks"`
	ConnectivityStatus    string     `json:"connectivity_status"`
	LastCheckedAt         *time.Time `json:"last_checked_at"`
	LastLatencyMS         int64      `json:"last_latency_ms"`
	LastConnectionError   string     `json:"last_connection_error"`
	CredentialsConfigured bool       `json:"credentials_configured"`
}

func networkRouterDTO(row models.NetworkRouter) networkRouterResponse {
	dto := networkRouterResponse{ID: row.ID, Code: row.Code, Name: row.Name, POPID: row.POPID, Host: row.Host, APIPort: row.APIPort, APIUsername: row.APIUsername, UseTLS: row.UseTLS, Status: row.Status, Remarks: row.Remarks, ConnectivityStatus: row.ConnectivityStatus, LastCheckedAt: row.LastCheckedAt, LastLatencyMS: row.LastLatencyMS, LastConnectionError: row.LastConnectionError, CredentialsConfigured: row.APIPasswordEncrypted != ""}
	if row.POP != nil {
		dto.POPName = row.POP.Name
	}
	return dto
}

type networkRouterCredentialRequest struct {
	Password string `json:"password" binding:"required"`
}

func SetNetworkRouterCredentials(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid router ID"})
			return
		}
		var req networkRouterCredentialRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		row, err := services.SetNetworkRouterPassword(uint(id), req.Password, cfg.RouterCredentialKey)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, networkRouterDTO(*row))
	}
}

func GetNetworkRouters(c *gin.Context) {
	rows, err := services.ListNetworkRouters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load routers"})
		return
	}
	response := make([]networkRouterResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, networkRouterDTO(row))
	}
	c.JSON(http.StatusOK, gin.H{"routers": response})
}

func CreateNetworkRouter(c *gin.Context) {
	var req networkRouterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row := models.NetworkRouter{Code: req.Code, Name: req.Name, POPID: req.POPID, Host: req.Host, APIPort: req.APIPort, APIUsername: req.APIUsername, UseTLS: req.UseTLS, Status: req.Status, Remarks: req.Remarks}
	if err := services.SaveNetworkRouter(&row); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	loaded, _ := services.GetNetworkRouter(row.ID)
	c.JSON(http.StatusCreated, networkRouterDTO(*loaded))
}

func UpdateNetworkRouter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid router ID"})
		return
	}
	row, err := services.GetNetworkRouter(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}
	var req networkRouterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	endpointChanged := row.Host != req.Host || row.APIPort != req.APIPort
	row.Code = req.Code
	row.Name = req.Name
	row.POPID = req.POPID
	row.Host = req.Host
	row.APIPort = req.APIPort
	row.APIUsername = req.APIUsername
	row.UseTLS = req.UseTLS
	row.Status = req.Status
	row.Remarks = req.Remarks
	if endpointChanged {
		row.ConnectivityStatus = "UNKNOWN"
		row.LastCheckedAt = nil
		row.LastLatencyMS = 0
		row.LastConnectionError = ""
	}
	if err := services.SaveNetworkRouter(row); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	loaded, _ := services.GetNetworkRouter(row.ID)
	c.JSON(http.StatusOK, networkRouterDTO(*loaded))
}

func TestNetworkRouterConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid router ID"})
		return
	}
	row, err := services.TestNetworkRouterConnection(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, networkRouterDTO(*row))
}
