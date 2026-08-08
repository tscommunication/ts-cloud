package services

import (
	"log"
	"time"
)

var ftpMonitorRunning bool

func StartFTPMonitor() {

	if ftpMonitorRunning {
		return
	}

	ftpMonitorRunning = true

	go func() {

		log.Println("FTP Monitor started")

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {

			if err := ProcessVSFTPDLogs(); err != nil {
				log.Println("FTP Monitor:", err)
			}

			<-ticker.C
		}
	}()
}
