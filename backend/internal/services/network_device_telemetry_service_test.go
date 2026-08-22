package services

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
)

func setupNetworkDeviceTelemetryServiceTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			filepath.Join(t.TempDir(), "telemetry.db"),
		),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.NetworkDevicePort{},
		&models.NetworkDevicePortSample{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func TestPersistNetworkDevicePortCandidatesFirstSample(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	sampledAt := time.Date(
		2026,
		time.August,
		23,
		6,
		0,
		0,
		0,
		time.UTC,
	)

	candidates := []snmpmonitor.PortPersistenceCandidate{
		{
			PortKey:     "ifindex:8",
			IfIndex:     8,
			Name:        "eth0/0/8",
			Description: "Wabda_OLT-UpLink",
			PortType:    "ETHERNET",
			AdminStatus: "UP",
			OperStatus:  "UP",
			SpeedMbps:   1000,
			InOctets:    1_000_000,
			OutOctets:   2_000_000,
			SampledAt:   sampledAt,
		},
	}

	if err := PersistNetworkDevicePortCandidates(
		1,
		candidates,
	); err != nil {
		t.Fatal(err)
	}

	var port models.NetworkDevicePort

	if err := db.Where(
		"network_device_id = ? AND port_key = ?",
		1,
		"ifindex:8",
	).First(&port).Error; err != nil {
		t.Fatal(err)
	}

	var sample models.NetworkDevicePortSample

	if err := db.Where(
		"network_device_port_id = ?",
		port.ID,
	).First(&sample).Error; err != nil {
		t.Fatal(err)
	}

	if sample.InMbps != 0 ||
		sample.OutMbps != 0 {
		t.Fatalf(
			"first sample rates must be zero: in=%f out=%f",
			sample.InMbps,
			sample.OutMbps,
		)
	}
}

func TestPersistNetworkDevicePortCandidatesCalculatesRates(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	base := time.Date(
		2026,
		time.August,
		23,
		6,
		0,
		0,
		0,
		time.UTC,
	)

	first := []snmpmonitor.PortPersistenceCandidate{
		{
			PortKey:     "ifindex:1",
			IfIndex:     1,
			Name:        "eth0/0/1",
			PortType:    "ETHERNET",
			AdminStatus: "UP",
			OperStatus:  "UP",
			InOctets:    1_000_000,
			OutOctets:   3_000_000,
			SampledAt:   base,
		},
	}

	second := []snmpmonitor.PortPersistenceCandidate{
		{
			PortKey:     "ifindex:1",
			IfIndex:     1,
			Name:        "eth0/0/1",
			PortType:    "ETHERNET",
			AdminStatus: "UP",
			OperStatus:  "UP",
			InOctets:    2_000_000,
			OutOctets:   5_000_000,
			SampledAt:   base.Add(10 * time.Second),
		},
	}

	if err := PersistNetworkDevicePortCandidates(
		1,
		first,
	); err != nil {
		t.Fatal(err)
	}

	if err := PersistNetworkDevicePortCandidates(
		1,
		second,
	); err != nil {
		t.Fatal(err)
	}

	var port models.NetworkDevicePort

	if err := db.Where(
		"network_device_id = ? AND port_key = ?",
		1,
		"ifindex:1",
	).First(&port).Error; err != nil {
		t.Fatal(err)
	}

	var samples []models.NetworkDevicePortSample

	if err := db.Where(
		"network_device_port_id = ?",
		port.ID,
	).Order(
		"sampled_at ASC",
	).Find(&samples).Error; err != nil {
		t.Fatal(err)
	}

	if len(samples) != 2 {
		t.Fatalf(
			"expected 2 samples, got %d",
			len(samples),
		)
	}

	if math.Abs(samples[1].InMbps-0.8) > 0.000001 {
		t.Fatalf(
			"expected in rate 0.8 got %.6f",
			samples[1].InMbps,
		)
	}

	if math.Abs(samples[1].OutMbps-1.6) > 0.000001 {
		t.Fatalf(
			"expected out rate 1.6 got %.6f",
			samples[1].OutMbps,
		)
	}
}

func TestPersistNetworkDevicePortCandidatesDuplicateTimestampIsIdempotent(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	now := time.Now()

	candidate := []snmpmonitor.PortPersistenceCandidate{
		{
			PortKey:     "ifindex:2",
			IfIndex:     2,
			Name:        "eth0/0/2",
			PortType:    "ETHERNET",
			AdminStatus: "UP",
			OperStatus:  "UP",
			InOctets:    100,
			OutOctets:   200,
			SampledAt:   now,
		},
	}

	if err := PersistNetworkDevicePortCandidates(
		1,
		candidate,
	); err != nil {
		t.Fatal(err)
	}

	if err := PersistNetworkDevicePortCandidates(
		1,
		candidate,
	); err != nil {
		t.Fatal(err)
	}

	var count int64

	if err := db.Model(
		&models.NetworkDevicePortSample{},
	).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"expected 1 sample, got %d",
			count,
		)
	}
}

func TestPersistNetworkDevicePortCandidatesRejectsOutOfOrderAndRollsBack(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	base := time.Now()

	first := []snmpmonitor.PortPersistenceCandidate{
		{
			PortKey:     "ifindex:3",
			IfIndex:     3,
			Name:        "eth0/0/3",
			PortType:    "ETHERNET",
			AdminStatus: "UP",
			OperStatus:  "UP",
			InOctets:    500,
			OutOctets:   600,
			SampledAt:   base,
		},
	}

	if err := PersistNetworkDevicePortCandidates(
		1,
		first,
	); err != nil {
		t.Fatal(err)
	}

	outOfOrder := []snmpmonitor.PortPersistenceCandidate{
		{
			PortKey:     "ifindex:3",
			IfIndex:     3,
			Name:        "eth0/0/3",
			Description: "must rollback",
			PortType:    "ETHERNET",
			AdminStatus: "UP",
			OperStatus:  "DOWN",
			InOctets:    100,
			OutOctets:   200,
			SampledAt:   base.Add(-time.Second),
		},
	}

	if err := PersistNetworkDevicePortCandidates(
		1,
		outOfOrder,
	); err == nil {
		t.Fatal("expected out-of-order error")
	}

	var port models.NetworkDevicePort

	if err := db.Where(
		"network_device_id = ? AND port_key = ?",
		1,
		"ifindex:3",
	).First(&port).Error; err != nil {
		t.Fatal(err)
	}

	if port.Description == "must rollback" {
		t.Fatal("port update was not rolled back")
	}

	var count int64

	if err := db.Model(
		&models.NetworkDevicePortSample{},
	).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"expected 1 sample after rollback, got %d",
			count,
		)
	}
}
