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
