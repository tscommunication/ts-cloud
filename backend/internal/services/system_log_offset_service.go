package services

import (
	"errors"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func GetOrCreateLogOffset(
	service string,
	logFile string,
) (*models.SystemLogOffset, error) {

	offset, err := repositories.GetSystemLogOffset(service)

	if err == nil {
		return offset, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	offset = &models.SystemLogOffset{
		ServiceName: service,
		LogFile:     logFile,
		LastOffset:  0,
		Inode:       0,
	}

	if err := repositories.CreateSystemLogOffset(offset); err != nil {
		return nil, err
	}

	return offset, nil
}

func SaveLogOffset(offset *models.SystemLogOffset) error {
	return repositories.SaveSystemLogOffset(offset)
}
