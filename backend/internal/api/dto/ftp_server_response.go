package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type FTPServerResponse struct {
	ID uint `json:"id"`

	Name string `json:"name"`

	Driver string `json:"driver"`

	Host string `json:"host"`

	Port int `json:"port"`

	Username string `json:"username"`

	RootPath string `json:"root_path"`

	PassivePortStart int `json:"passive_port_start"`

	PassivePortEnd int `json:"passive_port_end"`

	MaxConnections int `json:"max_connections"`

	Status string `json:"status"`

	Description string `json:"description"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

func ToFTPServerResponse(server models.FTPServer) FTPServerResponse {

	return FTPServerResponse{
		ID:                server.ID,
		Name:              server.Name,
		Driver:            server.Driver,
		Host:              server.Host,
		Port:              server.Port,
		Username:          server.Username,
		RootPath:          server.RootPath,
		PassivePortStart:  server.PassivePortStart,
		PassivePortEnd:    server.PassivePortEnd,
		MaxConnections:    server.MaxConnections,
		Status:            server.Status,
		Description:       server.Description,
		CreatedAt:         server.CreatedAt,
		UpdatedAt:         server.UpdatedAt,
	}
}
