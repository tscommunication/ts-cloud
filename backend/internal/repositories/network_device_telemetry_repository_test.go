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

func setupNetworkDeviceTelemetryRepositoryTestDB(
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

func TestUpsertNetworkDevicePortTxCreatesAndUpdates(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryRepositoryTestDB(t)

	now := time.Date(
		2026,
		time.August,
		23,
		5,
		50,
		0,
		0,
		time.UTC,
	)

	ifIndex := 8

	row := models.NetworkDevicePort{
		NetworkDeviceID: 1,
		PortKey:         "ifindex:8",
		IfIndex:         &ifIndex,
		Name:            "eth0/0/8",
		Description:     "OLT",
		PortType:        "ETHERNET",
		AdminStatus:     "UP",
		OperStatus:      "UP",
		SpeedMbps:       1000,
		LastSeenAt:      &now,
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return UpsertNetworkDevicePortTx(
			tx,
			&row,
		)
	}); err != nil {
		t.Fatal(err)
	}

	if row.ID == 0 {
		t.Fatal("expected created port ID")
	}

	createdID := row.ID

	row.Description = "Wabda_OLT-UpLink"
	row.OperStatus = "DOWN"
	row.SpeedMbps = 2500

	if err := db.Transaction(func(tx *gorm.DB) error {
		return UpsertNetworkDevicePortTx(
			tx,
			&row,
		)
	}); err != nil {
		t.Fatal(err)
	}

	if row.ID != createdID {
		t.Fatalf(
			"port ID changed from %d to %d",
			createdID,
			row.ID,
		)
	}

	var count int64

	if err := db.Model(
		&models.NetworkDevicePort{},
	).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"expected 1 port row, got %d",
			count,
		)
	}

	var saved models.NetworkDevicePort

	if err := db.First(
		&saved,
		createdID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if saved.Description != "Wabda_OLT-UpLink" {
		t.Fatalf(
			"unexpected description %q",
			saved.Description,
		)
	}

	if saved.OperStatus != "DOWN" {
		t.Fatalf(
			"unexpected oper status %q",
			saved.OperStatus,
		)
	}

	if saved.SpeedMbps != 2500 {
		t.Fatalf(
			"unexpected speed %d",
			saved.SpeedMbps,
		)
	}
}

func TestLatestNetworkDevicePortSampleTxReturnsLatest(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryRepositoryTestDB(t)

	base := time.Date(
		2026,
		time.August,
		23,
		5,
		50,
		0,
		0,
		time.UTC,
	)

	samples := []models.NetworkDevicePortSample{
		{
			NetworkDevicePortID: 10,
			SampledAt:           base,
			InOctets:            100,
		},
		{
			NetworkDevicePortID: 10,
			SampledAt: base.Add(
				5 * time.Minute,
			),
			InOctets: 200,
		},
	}

	for index := range samples {
		if err := db.Create(
			&samples[index],
		).Error; err != nil {
			t.Fatal(err)
		}
	}

	var latest *models.NetworkDevicePortSample

	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error

		latest, err =
			LatestNetworkDevicePortSampleTx(
				tx,
				10,
			)

		return err
	}); err != nil {
		t.Fatal(err)
	}

	if latest == nil {
		t.Fatal("expected latest sample")
	}

	if latest.InOctets != 200 {
		t.Fatalf(
			"expected latest in octets 200, got %d",
			latest.InOctets,
		)
	}
}

func TestLatestNetworkDevicePortSampleTxMissingReturnsNil(
	t *testing.T,
) {
	db := setupNetworkDeviceTelemetryRepositoryTestDB(t)

	var latest *models.NetworkDevicePortSample

	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error

		latest, err =
			LatestNetworkDevicePortSampleTx(
				tx,
				999,
			)

		return err
	}); err != nil {
		t.Fatal(err)
	}

	if latest != nil {
		t.Fatal("expected nil latest sample")
	}
}

func TestCreateNetworkDevicePortSampleTx(t *testing.T) {
	db := setupNetworkDeviceTelemetryRepositoryTestDB(t)

	row := models.NetworkDevicePortSample{
		NetworkDevicePortID: 7,
		SampledAt: time.Date(
			2026,
			time.August,
			23,
			6,
			0,
			0,
			0,
			time.UTC,
		),
		InOctets:  1000,
		OutOctets: 2000,
		InMbps:    1.25,
		OutMbps:   2.50,
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return CreateNetworkDevicePortSampleTx(
			tx,
			&row,
		)
	}); err != nil {
		t.Fatal(err)
	}

	if row.ID == 0 {
		t.Fatal("expected sample ID")
	}

	var saved models.NetworkDevicePortSample

	if err := db.First(
		&saved,
		row.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if saved.InOctets != 1000 ||
		saved.OutOctets != 2000 {
		t.Fatal("unexpected saved octets")
	}

	if saved.InMbps != 1.25 ||
		saved.OutMbps != 2.50 {
		t.Fatal("unexpected saved rates")
	}
}
