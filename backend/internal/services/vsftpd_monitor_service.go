package services

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/automation/linux"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func ProcessVSFTPDLogs() error {

	offset, err := GetOrCreateLogOffset(
		"vsftpd",
		linux.VSFTPDLogFile,
	)
	if err != nil {
		return err
	}

	lines, newOffset, inode, err := linux.ReadNewVSFTPDLogLines(
		offset.LastOffset,
		offset.Inode,
	)
	if err != nil {
		return err
	}

	for _, line := range lines {

		event := linux.ParseVSFTPDLine(line)

		if event == nil {
			continue
		}

		if err := processVSFTPDEvent(event); err != nil {
			continue
		}
	}

	offset.LastOffset = newOffset
	offset.Inode = inode

	return SaveLogOffset(offset)
}

func processVSFTPDEvent(
	event *linux.VSFTPDEvent,
) error {
	if event == nil {
		return nil
	}

	// Failed authentication attempts must be recorded even when the username
	// is not an active FTP account. Those events are security-relevant and the
	// FTPLoginLog model deliberately supports a nil FTPUserID for this case.
	if event.Type == linux.EventLoginFailed {
		return CreateFTPLoginLog(
			0,
			event.Username,
			event.IP,
			"FAILED",
			"vsftpd",
		)
	}

	user, err := repositories.GetFTPUserByUsername(
		event.Username,
	)
	if err != nil {

		// Unknown FTP user ? ignore
		return nil
	}

	switch event.Type {

	case linux.EventLoginSuccess:

		now := time.Now()

		user.LastLogin = &now
		user.LastIP = event.IP

		_ = repositories.UpdateFTPUser(user)

		_ = CreateFTPLoginLog(
			user.ID,
			user.Username,
			event.IP,
			"SUCCESS",
			"vsftpd",
		)

	case linux.EventUpload:

		user.TotalUploadBytes += uint64(event.Bytes)

		_ = repositories.UpdateFTPUser(user)

		_ = repositories.CreateFTPTransferLog(&models.FTPTransferLog{
			FTPUserID:    user.ID,
			Username:     user.Username,
			TransferType: "UPLOAD",
			Filename:     event.FileName,
			FileSize:     int64(event.Bytes),
			IPAddress:    event.IP,
			TransferTime: time.Now(),
		})

	case linux.EventDownload:

		user.TotalDownloadBytes += uint64(event.Bytes)

		_ = repositories.UpdateFTPUser(user)

		_ = repositories.CreateFTPTransferLog(&models.FTPTransferLog{
			FTPUserID:    user.ID,
			Username:     user.Username,
			TransferType: "DOWNLOAD",
			Filename:     event.FileName,
			FileSize:     int64(event.Bytes),
			IPAddress:    event.IP,
			TransferTime: time.Now(),
		})
	}

	return nil
}
