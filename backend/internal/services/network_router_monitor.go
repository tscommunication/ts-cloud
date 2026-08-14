package services

import (
	"log"
	"strings"
	"time"

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
		if _, err := SyncNetworkRouterResource(router.ID, keyMaterial); err != nil {
			log.Printf("Router monitor: router=%s sync failed: %v", router.Code, err)
		}
	}
}
