package services

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/mikrotik"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func ListNetworkRouters() ([]models.NetworkRouter, error) { return repositories.ListNetworkRouters() }
func GetNetworkRouter(id uint) (*models.NetworkRouter, error) {
	return repositories.GetNetworkRouter(id)
}

func SetNetworkRouterPassword(id uint, password, keyMaterial string) (*models.NetworkRouter, error) {
	row, err := repositories.GetNetworkRouter(id)
	if err != nil {
		return nil, errors.New("router not found")
	}
	encrypted, err := security.EncryptSecret(password, keyMaterial)
	if err != nil {
		return nil, err
	}
	row.APIPasswordEncrypted = encrypted
	row.APIStatus = "UNKNOWN"
	row.LastAuthenticatedAt = nil
	row.LastAPIError = ""
	row.RouterIdentity = ""
	row.RouterOSVersion = ""
	row.BoardName = ""
	row.RouterUptime = ""
	row.CPULoad = 0
	row.TotalMemory = 0
	row.FreeMemory = 0
	if err := repositories.UpdateNetworkRouter(row); err != nil {
		return nil, err
	}
	return row, nil
}

func SaveNetworkRouter(row *models.NetworkRouter) error {
	row.Code = strings.ToUpper(strings.TrimSpace(row.Code))
	row.Name = strings.TrimSpace(row.Name)
	row.Host = strings.TrimSpace(row.Host)
	row.APIUsername = strings.TrimSpace(row.APIUsername)
	row.Status = strings.ToUpper(strings.TrimSpace(row.Status))
	if row.ConnectivityStatus == "" {
		row.ConnectivityStatus = "UNKNOWN"
	}
	if row.Code == "" || row.Name == "" || row.Host == "" || row.APIUsername == "" {
		return errors.New("code, name, host and API username are required")
	}
	if net.ParseIP(row.Host) == nil {
		if parsed, err := url.Parse("//" + row.Host); err != nil || parsed.Hostname() != row.Host || strings.ContainsAny(row.Host, " /:@") {
			return errors.New("host must be a valid IP address or hostname")
		}
	}
	if row.APIPort < 1 || row.APIPort > 65535 {
		return errors.New("API port must be between 1 and 65535")
	}
	if row.Status != "ACTIVE" && row.Status != "INACTIVE" && row.Status != "MAINTENANCE" {
		return errors.New("invalid router status")
	}
	if row.POPID != nil {
		if _, err := repositories.GetPOP(*row.POPID); err != nil {
			return errors.New("POP not found")
		}
	}
	if row.ID == 0 {
		return repositories.CreateNetworkRouter(row)
	}
	return repositories.UpdateNetworkRouter(row)
}

func TestNetworkRouterConnection(id uint) (*models.NetworkRouter, error) {
	row, err := repositories.GetNetworkRouter(id)
	if err != nil {
		return nil, errors.New("router not found")
	}
	started := time.Now()
	connection, dialErr := net.DialTimeout("tcp", net.JoinHostPort(row.Host, strconv.Itoa(row.APIPort)), 3*time.Second)
	checkedAt := time.Now()
	row.LastCheckedAt = &checkedAt
	row.LastLatencyMS = checkedAt.Sub(started).Milliseconds()
	if dialErr != nil {
		row.ConnectivityStatus = "OFFLINE"
		row.LastTCPError = truncateRouterError(dialErr.Error())
	} else {
		_ = connection.Close()
		row.ConnectivityStatus = "ONLINE"
		row.LastTCPError = ""
	}
	if err := repositories.UpdateNetworkRouter(row); err != nil {
		return nil, err
	}
	return row, nil
}

func SyncNetworkRouterResource(id uint, keyMaterial string) (*models.NetworkRouter, error) {
	row, err := repositories.GetNetworkRouter(id)
	if err != nil {
		return nil, errors.New("router not found")
	}
	if row.APIPasswordEncrypted == "" {
		return row, errors.New("router API credentials are not configured")
	}
	password, err := security.DecryptSecret(row.APIPasswordEncrypted, keyMaterial)
	if err != nil {
		return row, err
	}
	started := time.Now()
	resource, syncErr := mikrotik.FetchResource(row.Host, row.APIPort, row.UseTLS, row.APIUsername, password)
	checkedAt := time.Now()
	row.LastCheckedAt = &checkedAt
	row.LastLatencyMS = checkedAt.Sub(started).Milliseconds()
	if syncErr != nil {
		row.APIStatus = "AUTH_FAILED"
		row.LastAPIError = truncateRouterError(syncErr.Error())
	} else {
		row.ConnectivityStatus = "ONLINE"
		row.LastTCPError = ""
		row.APIStatus = "AUTHENTICATED"
		row.LastAuthenticatedAt = &checkedAt
		row.LastAPIError = ""
		row.RouterIdentity = resource.Identity
		row.RouterOSVersion = resource.Version
		row.BoardName = resource.BoardName
		row.RouterUptime = resource.Uptime
		row.CPULoad = resource.CPULoad
		row.TotalMemory = resource.TotalMemory
		row.FreeMemory = resource.FreeMemory
	}
	if err := repositories.UpdateNetworkRouter(row); err != nil {
		return nil, err
	}
	return row, syncErr
}

func truncateRouterError(message string) string {
	if len(message) > 500 {
		return message[:500]
	}
	return message
}
