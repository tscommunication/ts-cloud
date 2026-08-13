package services

import (
	"log"
	"time"
)

func StartInvoiceOverdueWorker() {
	go func() {
		processInvoiceOverdues()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			processInvoiceOverdues()
		}
	}()
}

func processInvoiceOverdues() {
	count, err := ProcessInvoiceOverdues(time.Now())
	if err != nil {
		log.Printf("Invoice overdue worker: %v", err)
		return
	}
	if count > 0 {
		log.Printf("Invoice overdue worker: marked %d invoice(s) overdue", count)
	}
}
