package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func StartNetworkRouterMonitor(keyMaterial string, interval time.Duration, cpuThreshold, memoryThreshold int) {
	if interval <= 0 {
		log.Print("Router monitor: disabled")
		return
	}
	if len(strings.TrimSpace(keyMaterial)) < 32 {
		log.Print("Router monitor: disabled because ROUTER_CREDENTIAL_KEY is not configured")
		return
	}
	go func() {
		monitorNetworkRouters(keyMaterial, cpuThreshold, memoryThreshold)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			monitorNetworkRouters(keyMaterial, cpuThreshold, memoryThreshold)
		}
	}()
	log.Printf("Router monitor: started (interval=%s)", interval)
}

func monitorNetworkRouters(keyMaterial string, cpuThreshold, memoryThreshold int) {
	routers, err := repositories.ListMonitoredNetworkRouters()
	if err != nil {
		log.Printf("Router monitor: load routers: %v", err)
		return
	}
	for _, router := range routers {
		updated, syncErr := SyncNetworkRouterResource(router.ID, keyMaterial)
		if syncErr != nil {
			log.Printf("Router monitor: router=%s sync failed: %v", router.Code, syncErr)
		}
		if updated != nil {
			if err := recordNetworkRouterHealth(updated, time.Now()); err != nil {
				log.Printf("Router monitor: router=%s history failed: %v", router.Code, err)
			}
			if err := evaluateNetworkRouterStateAlerts(updated, time.Now()); err != nil {
				log.Printf("Router monitor: router=%s state alert evaluation failed: %v", router.Code, err)
			}
			if updated.APIStatus == "AUTHENTICATED" {
				if err := evaluateNetworkRouterAlerts(updated, cpuThreshold, memoryThreshold, time.Now()); err != nil {
					log.Printf("Router monitor: router=%s alert evaluation failed: %v", router.Code, err)
				}
			}
		}
	}
	if err := repositories.DeleteNetworkRouterHealthBefore(time.Now().Add(-30 * 24 * time.Hour)); err != nil {
		log.Printf("Router monitor: history retention cleanup failed: %v", err)
	}
}

func evaluateNetworkRouterStateAlerts(router *models.NetworkRouter, observedAt time.Time) error {
	offline := router.ConnectivityStatus == "OFFLINE"
	offlineMessage := fmt.Sprintf("Router %s is unreachable", router.Code)
	if err := updateNetworkRouterStateAlert(router, "ROUTER_OFFLINE", "CRITICAL", offline, offlineMessage, observedAt); err != nil {
		return err
	}
	apiFailed := router.ConnectivityStatus == "ONLINE" && router.APIStatus == "AUTH_FAILED"
	apiMessage := fmt.Sprintf("Router %s authenticated API sync is failing", router.Code)
	return updateNetworkRouterStateAlert(router, "API_FAILURE", "WARNING", apiFailed, apiMessage, observedAt)
}

func updateNetworkRouterStateAlert(router *models.NetworkRouter, alertType, severity string, breached bool, message string, observedAt time.Time) error {
	active, err := repositories.ActiveNetworkRouterAlert(router.ID, alertType)
	if err != nil {
		return err
	}
	if !breached {
		if active == nil {
			return nil
		}
		active.Status = "RESOLVED"
		active.CurrentValue = 0
		active.LastObservedAt = observedAt
		active.ResolvedAt = &observedAt
		if err := repositories.SaveNetworkRouterAlert(active); err != nil {
			return err
		}
		return SyncNetworkAlertNotification(active, router)
	}
	if active == nil {
		active = &models.NetworkRouterAlert{RouterID: router.ID, Type: alertType, Severity: severity, Status: "ACTIVE", Threshold: 1, OpenedAt: observedAt}
	}
	active.Message = message
	active.CurrentValue = 1
	active.LastObservedAt = observedAt
	if err := repositories.SaveNetworkRouterAlert(active); err != nil {
		return err
	}
	return SyncNetworkAlertNotification(active, router)
}

func evaluateNetworkRouterAlerts(router *models.NetworkRouter, cpuThreshold, memoryThreshold int, observedAt time.Time) error {
	if err := updateNetworkRouterAlert(router, "HIGH_CPU", float64(router.CPULoad), float64(cpuThreshold), observedAt); err != nil {
		return err
	}
	memoryUsedPercent := 0.0
	if router.TotalMemory > 0 {
		memoryUsedPercent = float64(router.TotalMemory-router.FreeMemory) * 100 / float64(router.TotalMemory)
	}
	return updateNetworkRouterAlert(router, "HIGH_MEMORY", memoryUsedPercent, float64(memoryThreshold), observedAt)
}

func updateNetworkRouterAlert(router *models.NetworkRouter, alertType string, value, threshold float64, observedAt time.Time) error {
	active, err := repositories.ActiveNetworkRouterAlert(router.ID, alertType)
	if err != nil {
		return err
	}
	breached := value >= threshold
	if !breached {
		if active == nil {
			return nil
		}
		active.Status = "RESOLVED"
		active.CurrentValue = value
		active.LastObservedAt = observedAt
		active.ResolvedAt = &observedAt
		if err := repositories.SaveNetworkRouterAlert(active); err != nil {
			return err
		}
		return SyncNetworkAlertNotification(active, router)
	}
	message := fmt.Sprintf("Router %s %s is %.1f%% (threshold %.1f%%)", router.Code, strings.ToLower(strings.TrimPrefix(alertType, "HIGH_")), value, threshold)
	if active == nil {
		active = &models.NetworkRouterAlert{RouterID: router.ID, Type: alertType, Severity: "WARNING", Status: "ACTIVE", OpenedAt: observedAt}
	}
	active.Message = message
	active.CurrentValue = value
	active.Threshold = threshold
	active.LastObservedAt = observedAt
	if err := repositories.SaveNetworkRouterAlert(active); err != nil {
		return err
	}
	return SyncNetworkAlertNotification(active, router)
}

func recordNetworkRouterHealth(router *models.NetworkRouter, observedAt time.Time) error {
	latest, err := repositories.LatestNetworkRouterHealth(router.ID)
	if err != nil {
		return err
	}
	statusChanged := latest == nil || latest.ConnectivityStatus != router.ConnectivityStatus || latest.APIStatus != router.APIStatus || latest.TCPError != router.LastTCPError || latest.APIError != router.LastAPIError
	if !statusChanged && observedAt.Sub(latest.ObservedAt) < 5*time.Minute {
		return nil
	}
	return repositories.CreateNetworkRouterHealth(&models.NetworkRouterHealth{
		RouterID: router.ID, ObservedAt: observedAt, ConnectivityStatus: router.ConnectivityStatus,
		APIStatus: router.APIStatus, LatencyMS: router.LastLatencyMS, CPULoad: router.CPULoad,
		TotalMemory: router.TotalMemory, FreeMemory: router.FreeMemory, RouterUptime: router.RouterUptime,
		TCPError: router.LastTCPError, APIError: router.LastAPIError,
	})
}
