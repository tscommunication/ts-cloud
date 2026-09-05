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
		&models.NetworkDevice{},
		&models.NetworkDevicePort{},
		&models.NetworkDevicePortSample{},
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
		&models.Notification{},
		&models.User{},
		&models.AgentNetworkDevice{},
	); err != nil {
		t.Fatal(err)
	}

	device := models.NetworkDevice{
		Code:               "OLT-TEST-001",
		Name:               "Test OLT",
		DeviceType:         "OLT",
		Vendor:              "TEST",
		DeviceModel:        "Test Model",
		ManagementIP:       "192.0.2.1",
		MonitoringProtocol: "SNMP",
	}
	if err := db.Create(&device).Error; err != nil {
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

func TestPersistNetworkDevicePortCandidatesRejectsCounterAboveSignedBigInt(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	candidate := []snmpmonitor.PortPersistenceCandidate{
		{
			PortKey:     "ifindex:11",
			IfIndex:     11,
			Name:        "eth0/0/11",
			PortType:    "ETHERNET",
			AdminStatus: "UP",
			OperStatus:  "UP",
			InOctets:    uint64(1) << 63,
			OutOctets:   100,
			SampledAt:   time.Now(),
		},
	}

	err := PersistNetworkDevicePortCandidates(
		1,
		candidate,
	)
	if err == nil {
		t.Fatal(
			"expected signed BIGINT range validation error",
		)
	}

	var portCount int64

	if err := db.Model(
		&models.NetworkDevicePort{},
	).Count(&portCount).Error; err != nil {
		t.Fatal(err)
	}

	if portCount != 0 {
		t.Fatalf(
			"expected transaction rollback with 0 ports, got %d",
			portCount,
		)
	}

	var sampleCount int64

	if err := db.Model(
		&models.NetworkDevicePortSample{},
	).Count(&sampleCount).Error; err != nil {
		t.Fatal(err)
	}

	if sampleCount != 0 {
		t.Fatalf(
			"expected transaction rollback with 0 samples, got %d",
			sampleCount,
		)
	}
}

func TestValidatePortPersistenceCandidateDatabaseRangeAcceptsMaxSignedBigInt(
	t *testing.T,
) {
	candidate := snmpmonitor.PortPersistenceCandidate{
		IfIndex:     12,
		InOctets:    maxPostgresBigInt,
		OutOctets:   maxPostgresBigInt,
		InErrors:    maxPostgresBigInt,
		OutErrors:   maxPostgresBigInt,
		InDiscards:  maxPostgresBigInt,
		OutDiscards: maxPostgresBigInt,
	}

	if err := validatePortPersistenceCandidateDatabaseRange(
		candidate,
	); err != nil {
		t.Fatalf(
			"max signed BIGINT should be accepted: %v",
			err,
		)
	}
}

func TestPersistNetworkDeviceONUCandidatesFirstSample(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	if err := db.AutoMigrate(
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	sampledAt := time.Date(
		2026,
		time.August,
		23,
		7,
		50,
		0,
		0,
		time.UTC,
	)

	rx := -12.64
	txPower := 2.22

	candidates := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:       1,
			ONUNo:       2,
			IfIndex:     14,
			MACAddress:  "E0:67:B3:11:22:33",
			Description: "EPON01ONU2",
			OperStatus:  "UP",
			InOctets:    1_000_000,
			OutOctets:   2_000_000,
			RxPowerDBM:  &rx,
			TxPowerDBM:  &txPower,
			SampledAt:   sampledAt,
		},
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		candidates,
	); err != nil {
		t.Fatal(err)
	}

	var onu models.NetworkDeviceONU

	if err := db.Where(
		"network_device_id = ? AND pon_no = ? AND onu_no = ?",
		1,
		1,
		2,
	).First(&onu).Error; err != nil {
		t.Fatal(err)
	}

	if onu.MACAddress != "E0:67:B3:11:22:33" {
		t.Fatalf(
			"persisted MAC=%q want=%q",
			onu.MACAddress,
			"E0:67:B3:11:22:33",
		)
	}

	var sample models.NetworkDeviceONUSample

	if err := db.Where(
		"network_device_onu_id = ?",
		onu.ID,
	).First(&sample).Error; err != nil {
		t.Fatal(err)
	}

	if sample.InMbps != 0 ||
		sample.OutMbps != 0 {
		t.Fatalf(
			"first ONU sample rates must be zero: in=%f out=%f",
			sample.InMbps,
			sample.OutMbps,
		)
	}

	if sample.RxPowerDBM == nil ||
		*sample.RxPowerDBM != rx {
		t.Fatal(
			"unexpected persisted RX power",
		)
	}
}

func TestPersistNetworkDeviceONUCandidatesCalculatesRates(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	if err := db.AutoMigrate(
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	base := time.Date(
		2026,
		time.August,
		23,
		7,
		50,
		0,
		0,
		time.UTC,
	)

	first := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:      1,
			ONUNo:      2,
			IfIndex:    14,
			OperStatus: "UP",
			InOctets:   1_000_000,
			OutOctets:  3_000_000,
			SampledAt:  base,
		},
	}

	second := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:      1,
			ONUNo:      2,
			IfIndex:    14,
			OperStatus: "UP",
			InOctets:   2_000_000,
			OutOctets:  5_000_000,
			SampledAt: base.Add(
				10 * time.Second,
			),
		},
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		first,
	); err != nil {
		t.Fatal(err)
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		second,
	); err != nil {
		t.Fatal(err)
	}

	var onu models.NetworkDeviceONU

	if err := db.Where(
		"network_device_id = ? AND pon_no = ? AND onu_no = ?",
		1,
		1,
		2,
	).First(&onu).Error; err != nil {
		t.Fatal(err)
	}

	var samples []models.NetworkDeviceONUSample

	if err := db.Where(
		"network_device_onu_id = ?",
		onu.ID,
	).Order(
		"sampled_at ASC",
	).Find(&samples).Error; err != nil {
		t.Fatal(err)
	}

	if len(samples) != 2 {
		t.Fatalf(
			"ONU sample count=%d want=2",
			len(samples),
		)
	}

	if math.Abs(
		samples[1].InMbps-0.8,
	) > 0.000001 {
		t.Fatalf(
			"in Mbps=%f want=0.8",
			samples[1].InMbps,
		)
	}

	if math.Abs(
		samples[1].OutMbps-1.6,
	) > 0.000001 {
		t.Fatalf(
			"out Mbps=%f want=1.6",
			samples[1].OutMbps,
		)
	}
}

func TestPersistNetworkDeviceONUCandidatesDuplicateTimestampIsIdempotent(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	if err := db.AutoMigrate(
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	candidates := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:      2,
			ONUNo:      12,
			IfIndex:    80,
			OperStatus: "UP",
			InOctets:   100,
			OutOctets:  200,
			SampledAt:  now,
		},
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		candidates,
	); err != nil {
		t.Fatal(err)
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		candidates,
	); err != nil {
		t.Fatal(err)
	}

	var count int64

	if err := db.Model(
		&models.NetworkDeviceONUSample{},
	).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"ONU sample count=%d want=1",
			count,
		)
	}
}

func TestPersistNetworkDeviceONUCandidatesRejectsOutOfOrderAndRollsBack(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	if err := db.AutoMigrate(
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	base := time.Now()

	first := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:       3,
			ONUNo:       10,
			IfIndex:     150,
			Description: "original",
			OperStatus:  "UP",
			InOctets:    500,
			OutOctets:   600,
			SampledAt:   base,
		},
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		first,
	); err != nil {
		t.Fatal(err)
	}

	outOfOrder := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:       3,
			ONUNo:       10,
			IfIndex:     150,
			Description: "must rollback",
			OperStatus:  "DOWN",
			InOctets:    100,
			OutOctets:   200,
			SampledAt: base.Add(
				-time.Second,
			),
		},
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		outOfOrder,
	); err == nil {
		t.Fatal(
			"expected out-of-order ONU error",
		)
	}

	var onu models.NetworkDeviceONU

	if err := db.Where(
		"network_device_id = ? AND pon_no = ? AND onu_no = ?",
		1,
		3,
		10,
	).First(&onu).Error; err != nil {
		t.Fatal(err)
	}

	if onu.Description == "must rollback" ||
		onu.OperStatus == "DOWN" {
		t.Fatal(
			"ONU update was not rolled back",
		)
	}

	var count int64

	if err := db.Model(
		&models.NetworkDeviceONUSample{},
	).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"ONU sample count=%d want=1 after rollback",
			count,
		)
	}
}

func TestPersistNetworkDeviceONUCandidatesRejectsCounterAboveSignedBigInt(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	if err := db.AutoMigrate(
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	candidates := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:      1,
			ONUNo:      31,
			IfIndex:    50,
			OperStatus: "UP",
			InOctets:   uint64(1) << 63,
			OutOctets:  100,
			SampledAt:  time.Now(),
		},
	}

	err := PersistNetworkDeviceONUCandidates(
		1,
		candidates,
	)

	if err == nil {
		t.Fatal(
			"expected signed BIGINT range validation error",
		)
	}

	var onuCount int64

	if err := db.Model(
		&models.NetworkDeviceONU{},
	).Count(&onuCount).Error; err != nil {
		t.Fatal(err)
	}

	if onuCount != 0 {
		t.Fatalf(
			"ONU count=%d want=0 after rollback",
			onuCount,
		)
	}

	var sampleCount int64

	if err := db.Model(
		&models.NetworkDeviceONUSample{},
	).Count(&sampleCount).Error; err != nil {
		t.Fatal(err)
	}

	if sampleCount != 0 {
		t.Fatalf(
			"ONU sample count=%d want=0 after rollback",
			sampleCount,
		)
	}
}

func TestPersistNetworkDeviceONUCandidatesPersistsAndPreservesRegistrationTimes(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryServiceTestDB(t)

	if err := db.AutoMigrate(
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	location := time.FixedZone(
		"Asia/Dhaka",
		6*60*60,
	)

	sampledAt := time.Date(
		2026,
		time.August,
		26,
		11,
		40,
		0,
		0,
		location,
	)

	lastRegistered := time.Date(
		2026,
		time.August,
		26,
		10,
		56,
		39,
		0,
		location,
	)

	lastDeregistered := time.Date(
		2026,
		time.August,
		26,
		9,
		58,
		51,
		0,
		location,
	)

	first := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:              1,
			ONUNo:              11,
			OperStatus:         "DOWN",
			LastRegisteredAt:   &lastRegistered,
			LastDeregisteredAt: &lastDeregistered,
			SampledAt:          sampledAt,
		},
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		first,
	); err != nil {
		t.Fatal(err)
	}

	var onu models.NetworkDeviceONU

	if err := db.Where(
		"network_device_id = ? AND pon_no = ? AND onu_no = ?",
		1,
		1,
		11,
	).First(&onu).Error; err != nil {
		t.Fatal(err)
	}

	if onu.LastRegisteredAt == nil ||
		!onu.LastRegisteredAt.Equal(lastRegistered) {
		t.Fatal(
			"last registered timestamp was not persisted",
		)
	}

	if onu.LastDeregisteredAt == nil ||
		!onu.LastDeregisteredAt.Equal(lastDeregistered) {
		t.Fatal(
			"last deregistered timestamp was not persisted",
		)
	}

	second := []snmpmonitor.ONUPersistenceCandidate{
		{
			PONNo:      1,
			ONUNo:      11,
			OperStatus: "UP",
			SampledAt: sampledAt.Add(
				time.Minute,
			),
		},
	}

	if err := PersistNetworkDeviceONUCandidates(
		1,
		second,
	); err != nil {
		t.Fatal(err)
	}

	if err := db.Where(
		"network_device_id = ? AND pon_no = ? AND onu_no = ?",
		1,
		1,
		11,
	).First(&onu).Error; err != nil {
		t.Fatal(err)
	}

	if onu.LastRegisteredAt == nil ||
		!onu.LastRegisteredAt.Equal(lastRegistered) {
		t.Fatal(
			"nil registration metadata erased known timestamp",
		)
	}

	if onu.LastDeregisteredAt == nil ||
		!onu.LastDeregisteredAt.Equal(lastDeregistered) {
		t.Fatal(
			"nil deregistration metadata erased known timestamp",
		)
	}
}
