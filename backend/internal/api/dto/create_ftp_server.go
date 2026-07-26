package dto

type CreateFTPServerRequest struct {
	Name string `json:"name" binding:"required"`

	Driver string `json:"driver"`

	Host string `json:"host" binding:"required"`

	Port int `json:"port"`

	Username string `json:"username" binding:"required"`

	Password string `json:"password" binding:"required"`

	RootPath string `json:"root_path" binding:"required"`

	PassivePortStart int `json:"passive_port_start"`

	PassivePortEnd int `json:"passive_port_end"`

	MaxConnections int `json:"max_connections"`

	Status string `json:"status"`

	Description string `json:"description"`
}
