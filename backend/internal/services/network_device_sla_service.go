package services

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type NetworkDeviceSLA struct {
	ID              uint    `json:"id"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	DeviceType      string  `json:"device_type"`
	Status          string  `json:"status"`
	Samples         int64   `json:"samples"`
	ObservedSeconds int64   `json:"observed_seconds"`
	DowntimeSeconds int64   `json:"downtime_seconds"`
	UptimePercent   float64 `json:"uptime_percent"`
}

func ListNetworkDeviceSLA(days int) ([]NetworkDeviceSLA, error) {
	return listNetworkDeviceSLAAt(days, time.Now())
}

func listNetworkDeviceSLAAt(days int, now time.Time) ([]NetworkDeviceSLA, error) {
	if days != 1 && days != 7 && days != 30 {
		days = 7
	}
	var devices []models.NetworkDevice
	if err := database.DB.Where("monitoring_enabled = ?", true).Order("device_type, code").Find(&devices).Error; err != nil {
		return nil, err
	}
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	rows := make([]NetworkDeviceSLA, 0, len(devices))
	for _, device := range devices {
		var previous models.NetworkDeviceHealth
		previousResult := database.DB.Where("network_device_id = ? AND observed_at < ?", device.ID, cutoff).Order("observed_at DESC").First(&previous)
		if previousResult.Error != nil && !errors.Is(previousResult.Error, gorm.ErrRecordNotFound) {
			return nil, previousResult.Error
		}

		var samples []models.NetworkDeviceHealth
		if err := database.DB.Where("network_device_id = ? AND observed_at >= ? AND observed_at <= ?", device.ID, cutoff, now).Order("observed_at").Find(&samples).Error; err != nil {
			return nil, err
		}
		if previousResult.Error == nil {
			samples = append([]models.NetworkDeviceHealth{previous}, samples...)
		}

		var observed, online int64
		for index, sample := range samples {
			start := sample.ObservedAt
			if start.Before(cutoff) {
				start = cutoff
			}
			end := now
			if index+1 < len(samples) {
				end = samples[index+1].ObservedAt
			}
			seconds := int64(end.Sub(start).Seconds())
			if seconds > 0 {
				observed += seconds
				if sample.Status == "ONLINE" {
					online += seconds
				}
			}
		}
		percent := 0.0
		if observed > 0 {
			percent = float64(online) * 100 / float64(observed)
		}
		rows = append(rows, NetworkDeviceSLA{
			ID: device.ID, Code: device.Code, Name: device.Name, DeviceType: device.DeviceType,
			Status: device.MonitoringStatus, Samples: int64(len(samples)), ObservedSeconds: observed,
			DowntimeSeconds: observed - online, UptimePercent: percent,
		})
	}
	return rows, nil
}
