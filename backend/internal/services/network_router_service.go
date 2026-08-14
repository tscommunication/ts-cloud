package services

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func ListNetworkRouters() ([]models.NetworkRouter, error) { return repositories.ListNetworkRouters() }
func GetNetworkRouter(id uint) (*models.NetworkRouter, error) {
	return repositories.GetNetworkRouter(id)
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
		row.LastConnectionError = dialErr.Error()
		if len(row.LastConnectionError) > 500 {
			row.LastConnectionError = row.LastConnectionError[:500]
		}
	} else {
		_ = connection.Close()
		row.ConnectivityStatus = "ONLINE"
		row.LastConnectionError = ""
	}
	if err := repositories.UpdateNetworkRouter(row); err != nil {
		return nil, err
	}
	return row, nil
}
