package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type FTPLoginLogResponse struct {
	ID uint `json:"id"`

	FTPUserID *uint `json:"ftp_user_id"`

	Username string `json:"username"`

	IPAddress string `json:"ip_address"`

	LoginStatus string `json:"login_status"`

	LoginTime time.Time `json:"login_time"`

	UserAgent string `json:"user_agent"`

	CreatedAt time.Time `json:"created_at"`
}

func ToFTPLoginLogResponse(
	log models.FTPLoginLog,
) FTPLoginLogResponse {

	return FTPLoginLogResponse{

		ID: log.ID,

		FTPUserID: log.FTPUserID,

		Username: log.Username,

		IPAddress: log.IPAddress,

		LoginStatus: log.LoginStatus,

		LoginTime: log.LoginTime,

		UserAgent: log.UserAgent,

		CreatedAt: log.CreatedAt,
	}
}
