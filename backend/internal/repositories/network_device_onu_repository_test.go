package repositories

import (
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupNetworkDeviceONURepositoryTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			filepath.Join(
				t.TempDir(),
				"onu-telemetry.db",
			),
		),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
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

func TestUpsertNetworkDeviceONUTxCreatesAndUpdates(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	now := time.Date(
		2026,
		time.August,
		23,
		7,
		40,
		0,
		0,
		time.UTC,
	)

	ifIndex := 14

	row := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           2,
		IfIndex:         &ifIndex,
		Description:     "EPON01ONU2",
		OperStatus:      "UP",
		LastSeenAt:      &now,
		UpdatedAt:       now,
	}

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			return UpsertNetworkDeviceONUTx(
				tx,
				&row,
			)
		},
	); err != nil {
		t.Fatal(err)
	}

	if row.ID == 0 {
		t.Fatal("expected created ONU ID")
	}

	createdID := row.ID

	row.OperStatus = "DOWN"
	row.Description = "updated ONU"

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			return UpsertNetworkDeviceONUTx(
				tx,
				&row,
			)
		},
	); err != nil {
		t.Fatal(err)
	}

	if row.ID != createdID {
		t.Fatalf(
			"ONU ID changed from %d to %d",
			createdID,
			row.ID,
		)
	}

	var count int64

	if err := db.Model(
		&models.NetworkDeviceONU{},
	).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"ONU count=%d want=1",
			count,
		)
	}

	if row.OperStatus != "DOWN" {
		t.Fatalf(
			"oper status=%q want=DOWN",
			row.OperStatus,
		)
	}
}

func TestLatestNetworkDeviceONUSampleTxReturnsLatest(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	base := time.Date(
		2026,
		time.August,
		23,
		7,
		40,
		0,
		0,
		time.UTC,
	)

	samples := []models.NetworkDeviceONUSample{
		{
			NetworkDeviceONUID: 10,
			SampledAt:          base,
			InOctets:           100,
		},
		{
			NetworkDeviceONUID: 10,
			SampledAt: base.Add(
				5 * time.Minute,
			),
			InOctets: 200,
		},
	}

	if err := db.Create(&samples).Error; err != nil {
		t.Fatal(err)
	}

	var latest *models.NetworkDeviceONUSample

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			var err error

			latest, err =
				LatestNetworkDeviceONUSampleTx(
					tx,
					10,
				)

			return err
		},
	); err != nil {
		t.Fatal(err)
	}

	if latest == nil {
		t.Fatal("expected latest ONU sample")
	}

	if latest.InOctets != 200 {
		t.Fatalf(
			"in octets=%d want=200",
			latest.InOctets,
		)
	}
}

func TestLatestNetworkDeviceONUOpticalSampleReturnsLatestValidOptical(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	base := time.Date(
		2026,
		time.August,
		23,
		7,
		55,
		0,
		0,
		time.UTC,
	)

	rxOld := -18.5
	txOld := 2.1
	rxNew := -17.25
	txNew := 2.25

	samples := []models.NetworkDeviceONUSample{
		{
			NetworkDeviceONUID: 20,
			SampledAt:          base,
			RxPowerDBM:         &rxOld,
			TxPowerDBM:         &txOld,
		},
		{
			NetworkDeviceONUID: 20,
			SampledAt: base.Add(
				5 * time.Minute,
			),
			RxPowerDBM: &rxNew,
			TxPowerDBM: &txNew,
		},
		{
			NetworkDeviceONUID: 20,
			SampledAt: base.Add(
				10 * time.Minute,
			),
			InMbps:  12.5,
			OutMbps: 3.25,
		},
	}

	if err := db.Create(&samples).Error; err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
	})

	got, err := LatestNetworkDeviceONUOpticalSample(20)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("expected optical sample")
	}

	if !got.SampledAt.Equal(base.Add(5 * time.Minute)) {
		t.Fatalf(
			"sampled_at=%s want=%s",
			got.SampledAt,
			base.Add(5*time.Minute),
		)
	}

	if got.RxPowerDBM == nil || *got.RxPowerDBM != rxNew {
		t.Fatalf(
			"rx_power_dbm=%v want=%v",
			got.RxPowerDBM,
			rxNew,
		)
	}

	if got.TxPowerDBM == nil || *got.TxPowerDBM != txNew {
		t.Fatalf(
			"tx_power_dbm=%v want=%v",
			got.TxPowerDBM,
			txNew,
		)
	}
}

func TestLatestNetworkDeviceONUOpticalSampleMissingReturnsNil(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	sample := models.NetworkDeviceONUSample{
		NetworkDeviceONUID: 21,
		SampledAt: time.Date(
			2026,
			time.August,
			23,
			8,
			0,
			0,
			0,
			time.UTC,
		),
		InMbps:  5,
		OutMbps: 2,
	}

	if err := db.Create(&sample).Error; err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
	})

	got, err := LatestNetworkDeviceONUOpticalSample(21)
	if err != nil {
		t.Fatal(err)
	}

	if got != nil {
		t.Fatalf(
			"expected nil optical sample, got=%+v",
			got,
		)
	}
}

func TestLatestNetworkDeviceONUSampleTxMissingReturnsNil(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	var latest *models.NetworkDeviceONUSample

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			var err error

			latest, err =
				LatestNetworkDeviceONUSampleTx(
					tx,
					999,
				)

			return err
		},
	); err != nil {
		t.Fatal(err)
	}

	if latest != nil {
		t.Fatal(
			"expected nil latest ONU sample",
		)
	}
}

func TestCreateNetworkDeviceONUSampleTx(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	rx := -12.64
	txPower := 2.22

	row := models.NetworkDeviceONUSample{
		NetworkDeviceONUID: 7,
		SampledAt: time.Date(
			2026,
			time.August,
			23,
			7,
			40,
			0,
			0,
			time.UTC,
		),
		InOctets:   1000,
		OutOctets:  2000,
		RxPowerDBM: &rx,
		TxPowerDBM: &txPower,
	}

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			return CreateNetworkDeviceONUSampleTx(
				tx,
				&row,
			)
		},
	); err != nil {
		t.Fatal(err)
	}

	if row.ID == 0 {
		t.Fatal("expected ONU sample ID")
	}

	var saved models.NetworkDeviceONUSample

	if err := db.First(
		&saved,
		row.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if saved.RxPowerDBM == nil ||
		*saved.RxPowerDBM != rx {
		t.Fatal(
			"unexpected saved RX power",
		)
	}

	if saved.TxPowerDBM == nil ||
		*saved.TxPowerDBM != txPower {
		t.Fatal(
			"unexpected saved TX power",
		)
	}
}

func TestListNetworkDeviceONUsOrdersByPONAndONU(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	rows := []models.NetworkDeviceONU{
		{
			NetworkDeviceID: 1,
			PONNo:           2,
			ONUNo:           1,
		},
		{
			NetworkDeviceID: 1,
			PONNo:           1,
			ONUNo:           10,
		},
		{
			NetworkDeviceID: 2,
			PONNo:           1,
			ONUNo:           1,
		},
		{
			NetworkDeviceID: 1,
			PONNo:           1,
			ONUNo:           2,
		},
	}

	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	got, err := ListNetworkDeviceONUs(1)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 3 {
		t.Fatalf(
			"ONU count=%d want=3",
			len(got),
		)
	}

	if got[0].PONNo != 1 ||
		got[0].ONUNo != 2 ||
		got[1].PONNo != 1 ||
		got[1].ONUNo != 10 ||
		got[2].PONNo != 2 ||
		got[2].ONUNo != 1 {
		t.Fatal(
			"unexpected ONU ordering",
		)
	}
}

func TestUpsertNetworkDeviceONUTelemetryTxPreservesInventory(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	base := time.Date(
		2026,
		time.August,
		23,
		8,
		0,
		0,
		0,
		time.UTC,
	)

	oldIfIndex := 14
	registeredAt := base.Add(-24 * time.Hour)
	deregisteredAt := base.Add(-12 * time.Hour)

	existing := models.NetworkDeviceONU{
		NetworkDeviceID:    1,
		PONNo:              1,
		ONUNo:              2,
		IfIndex:            &oldIfIndex,
		MACAddress:         "AA:BB:CC:DD:EE:FF",
		SerialNumber:       "VSOL-SERIAL-001",
		Model:              "V2801",
		Capability:         "EPON",
		Description:        "EPON01ONU2",
		OperStatus:         "UP",
		DistanceM:          1234,
		LastRegisteredAt:   &registeredAt,
		LastDeregisteredAt: &deregisteredAt,
		UptimeSeconds:      98765,
		LastSeenAt:         &base,
		CreatedAt:          base,
		UpdatedAt:          base,
	}

	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}

	next := base.Add(5 * time.Minute)

	telemetry := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           2,
		IfIndex:         nil,
		Description:     "",
		OperStatus:      "UNKNOWN",
		LastSeenAt:      &next,
		UpdatedAt:       next,
	}

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			return UpsertNetworkDeviceONUTelemetryTx(
				tx,
				&telemetry,
			)
		},
	); err != nil {
		t.Fatal(err)
	}

	if telemetry.ID != existing.ID {
		t.Fatalf(
			"ONU ID changed from %d to %d",
			existing.ID,
			telemetry.ID,
		)
	}

	if telemetry.MACAddress != existing.MACAddress ||
		telemetry.SerialNumber != existing.SerialNumber ||
		telemetry.Model != existing.Model ||
		telemetry.Capability != existing.Capability ||
		telemetry.DistanceM != existing.DistanceM ||
		telemetry.UptimeSeconds != existing.UptimeSeconds {
		t.Fatal(
			"telemetry upsert overwrote inventory-owned fields",
		)
	}

	if telemetry.LastRegisteredAt == nil ||
		!telemetry.LastRegisteredAt.Equal(registeredAt) ||
		telemetry.LastDeregisteredAt == nil ||
		!telemetry.LastDeregisteredAt.Equal(deregisteredAt) {
		t.Fatal(
			"telemetry upsert overwrote registration metadata",
		)
	}

	if telemetry.IfIndex == nil ||
		*telemetry.IfIndex != oldIfIndex {
		t.Fatal(
			"nil telemetry ifIndex cleared known ifIndex",
		)
	}

	if telemetry.Description != existing.Description {
		t.Fatalf(
			"description=%q want=%q",
			telemetry.Description,
			existing.Description,
		)
	}

	if telemetry.OperStatus != "UP" {
		t.Fatalf(
			"oper status=%q want=UP",
			telemetry.OperStatus,
		)
	}

	if telemetry.LastSeenAt == nil ||
		!telemetry.LastSeenAt.Equal(next) {
		t.Fatal(
			"telemetry last seen was not updated",
		)
	}
}

func TestUpsertNetworkDeviceONUTelemetryTxUpdatesKnownMAC(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	base := time.Date(
		2026,
		time.August,
		25,
		23,
		30,
		0,
		0,
		time.UTC,
	)

	existing := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           8,
		OperStatus:      "UP",
		LastSeenAt:      &base,
		CreatedAt:       base,
		UpdatedAt:       base,
	}

	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}

	next := base.Add(time.Minute)

	telemetry := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           8,
		MACAddress:      "E0:67:B3:11:22:33",
		LastSeenAt:      &next,
		UpdatedAt:       next,
	}

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			return UpsertNetworkDeviceONUTelemetryTx(
				tx,
				&telemetry,
			)
		},
	); err != nil {
		t.Fatal(err)
	}

	if telemetry.MACAddress != "E0:67:B3:11:22:33" {
		t.Fatalf(
			"MAC=%q want=%q",
			telemetry.MACAddress,
			"E0:67:B3:11:22:33",
		)
	}
}

func TestUpsertNetworkDeviceONUTelemetryTxBlankMACPreservesKnownMAC(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	base := time.Date(
		2026,
		time.August,
		25,
		23,
		31,
		0,
		0,
		time.UTC,
	)

	existing := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           9,
		MACAddress:      "AA:BB:CC:DD:EE:FF",
		OperStatus:      "UP",
		LastSeenAt:      &base,
		CreatedAt:       base,
		UpdatedAt:       base,
	}

	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}

	next := base.Add(time.Minute)

	telemetry := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           9,
		LastSeenAt:      &next,
		UpdatedAt:       next,
	}

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			return UpsertNetworkDeviceONUTelemetryTx(
				tx,
				&telemetry,
			)
		},
	); err != nil {
		t.Fatal(err)
	}

	if telemetry.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf(
			"MAC=%q want=%q",
			telemetry.MACAddress,
			"AA:BB:CC:DD:EE:FF",
		)
	}
}

func TestUpsertNetworkDeviceONUTelemetryTxUpdatesTelemetryFields(
	t *testing.T,
) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	base := time.Date(
		2026,
		time.August,
		23,
		8,
		10,
		0,
		0,
		time.UTC,
	)

	oldIfIndex := 14

	existing := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           3,
		IfIndex:         &oldIfIndex,
		SerialNumber:    "KEEP-SERIAL",
		Model:           "KEEP-MODEL",
		Description:     "old description",
		OperStatus:      "UP",
		LastSeenAt:      &base,
		CreatedAt:       base,
		UpdatedAt:       base,
	}

	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}

	newIfIndex := 15
	next := base.Add(5 * time.Minute)

	telemetry := models.NetworkDeviceONU{
		NetworkDeviceID: 1,
		PONNo:           1,
		ONUNo:           3,
		IfIndex:         &newIfIndex,
		Description:     "EPON01ONU3",
		OperStatus:      "DOWN",
		LastSeenAt:      &next,
		UpdatedAt:       next,
	}

	if err := db.Transaction(
		func(tx *gorm.DB) error {
			return UpsertNetworkDeviceONUTelemetryTx(
				tx,
				&telemetry,
			)
		},
	); err != nil {
		t.Fatal(err)
	}

	if telemetry.IfIndex == nil ||
		*telemetry.IfIndex != newIfIndex {
		t.Fatal(
			"telemetry ifIndex was not updated",
		)
	}

	if telemetry.Description != "EPON01ONU3" ||
		telemetry.OperStatus != "DOWN" {
		t.Fatal(
			"telemetry-owned fields were not updated",
		)
	}

	if telemetry.SerialNumber != "KEEP-SERIAL" ||
		telemetry.Model != "KEEP-MODEL" {
		t.Fatal(
			"inventory metadata was not preserved",
		)
	}
}

func TestFindNetworkDeviceONUByPosition(t *testing.T) {
	db := setupNetworkDeviceONURepositoryTestDB(t)

	ifIndex := 21495809
	now := time.Date(
		2026,
		time.September,
		1,
		13,
		4,
		10,
		0,
		time.UTC,
	)

	row := models.NetworkDeviceONU{
		NetworkDeviceID: 9,
		PONNo:           1,
		ONUNo:           1,
		IfIndex:         &ifIndex,
		MACAddress:      "70:A5:6A:0C:37:A2",
		Description:     "epon 0/1/1 onu 1",
		OperStatus:      "UP",
		LastSeenAt:      &now,
	}

	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	found, err := FindNetworkDeviceONUByPosition(
		9,
		1,
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil {
		t.Fatal("expected exact ONU")
	}

	if found.ID != row.ID {
		t.Fatalf(
			"ONU ID = %d, want %d",
			found.ID,
			row.ID,
		)
	}
	if found.NetworkDeviceID != 9 {
		t.Fatalf(
			"NetworkDeviceID = %d, want 9",
			found.NetworkDeviceID,
		)
	}
	if found.PONNo != 1 || found.ONUNo != 1 {
		t.Fatalf(
			"ONU position = PON %d ONU %d, want PON 1 ONU 1",
			found.PONNo,
			found.ONUNo,
		)
	}
	if found.MACAddress != "70:A5:6A:0C:37:A2" {
		t.Fatalf(
			"ONU MAC = %q, want expected ONU MAC",
			found.MACAddress,
		)
	}
}

func TestFindNetworkDeviceONUByPositionNotFound(t *testing.T) {
	setupNetworkDeviceONURepositoryTestDB(t)

	found, err := FindNetworkDeviceONUByPosition(
		9,
		1,
		99,
	)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatalf("found = %#v, want nil", found)
	}
}

func TestFindNetworkDeviceONUByPositionValidatesInput(t *testing.T) {
	setupNetworkDeviceONURepositoryTestDB(t)

	tests := []struct {
		name     string
		deviceID uint
		ponNo    int
		onuNo    int
	}{
		{
			name:     "missing device",
			deviceID: 0,
			ponNo:    1,
			onuNo:    1,
		},
		{
			name:     "invalid PON",
			deviceID: 9,
			ponNo:    0,
			onuNo:    1,
		},
		{
			name:     "invalid ONU",
			deviceID: 9,
			ponNo:    1,
			onuNo:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := FindNetworkDeviceONUByPosition(
				tt.deviceID,
				tt.ponNo,
				tt.onuNo,
			)
			if err == nil {
				t.Fatalf(
					"found = %#v, want validation error",
					found,
				)
			}
		})
	}
}
