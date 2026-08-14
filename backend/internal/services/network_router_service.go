package services

import (
	"errors"
	"net"
	"net/url"
	"strings"

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
