package services

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

const networkDeviceMonitorTick = 30 * time.Second

func StartNetworkDeviceMonitor(keyMaterial string) {
	if len(strings.TrimSpace(keyMaterial)) < 32 {
		log.Print("Network device monitor: disabled because ROUTER_CREDENTIAL_KEY is not configured")
		return
	}
	go func() {
		monitorNetworkDevices(keyMaterial, time.Now())
		ticker := time.NewTicker(networkDeviceMonitorTick)
		defer ticker.Stop()
		for observedAt := range ticker.C {
			monitorNetworkDevices(keyMaterial, observedAt)
		}
	}()
	log.Printf("Network device monitor: started (scheduler interval=%s)", networkDeviceMonitorTick)
}

func monitorNetworkDevices(keyMaterial string, observedAt time.Time) {
	var devices []models.NetworkDevice
	if err := database.DB.Where("monitoring_enabled = ?", true).Order("id").Find(&devices).Error; err != nil {
		log.Printf("Network device monitor: load devices: %v", err)
		return
	}

	semaphore := make(chan struct{}, 5)
	var waitGroup sync.WaitGroup
	for index := range devices {
		device := devices[index]
		if !networkDevicePollDue(&device, observedAt) {
			continue
		}
		waitGroup.Add(1)
		semaphore <- struct{}{}
		go func() {
			defer waitGroup.Done()
			defer func() { <-semaphore }()

			result, err := pollNetworkDeviceSNMPv2c(
				&device,
				keyMaterial,
				observedAt,
				defaultNetworkDevicePollDeps(),
			)
			if err != nil {
				log.Printf(
					"Network device monitor: device=%s poll failed: %v",
					device.Code,
					err,
				)
				if stateErr := recordNetworkDevicePollFailure(
					device.ID,
					observedAt,
					err,
				); stateErr != nil {
					log.Printf(
						"Network device monitor: device=%s failure state update failed: %v",
						device.Code,
						stateErr,
					)
				}
				return
			}

			lastError := ""
			joinedError := errors.Join(
				result.ProbeError,
				result.TelemetryError,
				result.ONUError,
			)
			if joinedError != nil {
				lastError = joinedError.Error()
			}

			if err := database.DB.Model(
				&models.NetworkDevice{},
			).Where(
				"id = ?",
				device.ID,
			).Updates(
				map[string]any{
					"monitoring_status": result.Status,
					"last_polled_at":    observedAt,
					"last_error":        lastError,
				},
			).Error; err != nil {
				log.Printf(
					"Network device monitor: device=%s state update failed: %v",
					device.Code,
					err,
				)
				return
			}

			if result.TelemetryError != nil {
				log.Printf(
					"Network device monitor: device=%s status=%s telemetry warning: %v",
					device.Code,
					result.Status,
					result.TelemetryError,
				)
			}

			if result.ONUError != nil {
				log.Printf(
					"Network device monitor: device=%s status=%s ONU warning: %v",
					device.Code,
					result.Status,
					result.ONUError,
				)
			}

			log.Printf(
				"Network device monitor: device=%s status=%s ports=%d onus=%d onu_adapter=%s",
				device.Code,
				result.Status,
				result.PortCount,
				result.ONUCount,
				result.ONUAdapter,
			)
		}()
	}
	waitGroup.Wait()
}

func recordNetworkDevicePollFailure(
	deviceID uint,
	observedAt time.Time,
	pollErr error,
) error {
	if deviceID == 0 {
		return errors.New("network device ID is required")
	}
	if observedAt.IsZero() {
		return errors.New("poll observation time is required")
	}
	if pollErr == nil {
		return errors.New("poll error is required")
	}

	return database.DB.Model(&models.NetworkDevice{}).
		Where("id = ?", deviceID).
		Updates(map[string]any{
			"monitoring_status": "OFFLINE",
			"last_polled_at":    observedAt,
			"last_error":        "poll: " + pollErr.Error(),
		}).Error
}

func networkDevicePollDue(device *models.NetworkDevice, observedAt time.Time) bool {
	if device == nil || !device.MonitoringEnabled {
		return false
	}
	if device.LastPolledAt == nil {
		return true
	}
	interval := time.Duration(device.PollingInterval) * time.Second
	if interval < networkDeviceMonitorTick {
		interval = networkDeviceMonitorTick
	}
	return !device.LastPolledAt.Add(interval).After(observedAt)
}
