package services

import (
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
			updated, err := TestNetworkDeviceConnection(device.ID, keyMaterial)
			if err != nil {
				log.Printf("Network device monitor: device=%s probe failed: %v", device.Code, err)
				return
			}
			log.Printf("Network device monitor: device=%s status=%s", updated.Code, updated.MonitoringStatus)
		}()
	}
	waitGroup.Wait()
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
