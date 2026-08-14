package services

import (
	"log"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func StartNetworkRouterMonitor(keyMaterial string, interval time.Duration) {
	if interval <= 0 {
		log.Print("Router monitor: disabled")
		return
	}
	if len(strings.TrimSpace(keyMaterial)) < 32 {
		log.Print("Router monitor: disabled because ROUTER_CREDENTIAL_KEY is not configured")
		return
	}
	go func() {
		monitorNetworkRouters(keyMaterial)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			monitorNetworkRouters(keyMaterial)
		}
	}()
	log.Printf("Router monitor: started (interval=%s)", interval)
}

func monitorNetworkRouters(keyMaterial string) {
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
		}
	}
	if err := repositories.DeleteNetworkRouterHealthBefore(time.Now().Add(-30 * 24 * time.Hour)); err != nil {
		log.Printf("Router monitor: history retention cleanup failed: %v", err)
	}
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
