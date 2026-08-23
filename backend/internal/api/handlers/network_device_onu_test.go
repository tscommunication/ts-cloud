package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupNetworkDeviceONUHandlerTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			filepath.Join(
				t.TempDir(),
				"network-device-onu-handler.db",
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
		&models.POP{},
		&models.NetworkRouter{},
		&models.NetworkDevice{},
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previous
	})

	return db
}

func newNetworkDeviceONUTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.GET(
		"/network/devices/:id/onus",
		ListNetworkDeviceONUs,
	)

	return router
}

func createNetworkDeviceONUHandlerTestDevice(
	t *testing.T,
	db *gorm.DB,
) models.NetworkDevice {
	t.Helper()

	device := models.NetworkDevice{
		Code:               "OLT-ONU-TEST",
		Name:               "ONU Test OLT",
		DeviceType:         "OLT",
		Vendor:             "VSOL",
		DeviceModel:        "V1601E04-DP",
		OLTType:            "EPON",
		ManagementIP:       "192.0.2.20",
		ManagementPort:     9001,
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
	}

	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	return device
}

func TestListNetworkDeviceONUsHandler(
	t *testing.T,
) {
	db := setupNetworkDeviceONUHandlerTestDB(t)
	device := createNetworkDeviceONUHandlerTestDevice(
		t,
		db,
	)

	ifIndex := 101
	now := time.Date(
		2026,
		time.August,
		23,
		8,
		0,
		0,
		0,
		time.UTC,
	)

	onu := models.NetworkDeviceONU{
		NetworkDeviceID: device.ID,
		PONNo:           1,
		ONUNo:           7,
		IfIndex:         &ifIndex,
		MACAddress:      "AA:BB:CC:DD:EE:77",
		Description:     "Customer Test",
		OperStatus:      "UP",
		DistanceM:       850,
		LastSeenAt:      &now,
	}

	if err := db.Create(&onu).Error; err != nil {
		t.Fatal(err)
	}

	temperature := 41.5
	voltage := 3.29
	txPower := 2.1
	rxPower := -18.7
	distance := 860

	sample := models.NetworkDeviceONUSample{
		NetworkDeviceONUID: onu.ID,
		SampledAt:          now,
		InOctets:           1000,
		OutOctets:          2000,
		InMbps:             12.5,
		OutMbps:            3.25,
		TemperatureC:       &temperature,
		VoltageV:           &voltage,
		TxPowerDBM:         &txPower,
		RxPowerDBM:         &rxPower,
		DistanceM:          &distance,
	}

	if err := db.Create(&sample).Error; err != nil {
		t.Fatal(err)
	}

	router := newNetworkDeviceONUTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices/"+itoaUint(device.ID)+"/onus",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body = %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response struct {
		ONUs []struct {
			ID           uint   `json:"id"`
			PONNo        int    `json:"pon_no"`
			ONUNo        int    `json:"onu_no"`
			MACAddress   string `json:"mac_address"`
			Description  string `json:"description"`
			OperStatus   string `json:"oper_status"`
			LatestSample *struct {
				InMbps       float64  `json:"in_mbps"`
				OutMbps      float64  `json:"out_mbps"`
				TemperatureC *float64 `json:"temperature_c"`
				VoltageV     *float64 `json:"voltage_v"`
				TxPowerDBM   *float64 `json:"tx_power_dbm"`
				RxPowerDBM   *float64 `json:"rx_power_dbm"`
				DistanceM    int      `json:"distance_m"`
			} `json:"latest_sample"`
		} `json:"onus"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if len(response.ONUs) != 1 {
		t.Fatalf(
			"ONU count = %d, want 1",
			len(response.ONUs),
		)
	}

	got := response.ONUs[0]

	if got.PONNo != 1 ||
		got.ONUNo != 7 ||
		got.MACAddress != "AA:BB:CC:DD:EE:77" ||
		got.Description != "Customer Test" ||
		got.OperStatus != "UP" {
		t.Fatalf(
			"unexpected ONU response: %+v",
			got,
		)
	}

	if got.LatestSample == nil {
		t.Fatal("latest_sample is nil")
	}

	if got.LatestSample.InMbps != 12.5 ||
		got.LatestSample.OutMbps != 3.25 ||
		got.LatestSample.DistanceM != 860 {
		t.Fatalf(
			"unexpected latest sample: %+v",
			got.LatestSample,
		)
	}
}

func TestListNetworkDeviceONUsHandlerUsesLatestValidOptical(
	t *testing.T,
) {
	db := setupNetworkDeviceONUHandlerTestDB(t)

	device := createNetworkDeviceONUHandlerTestDevice(
		t,
		db,
	)

	now := time.Date(
		2026,
		time.August,
		23,
		8,
		0,
		0,
		0,
		time.UTC,
	)

	onu := models.NetworkDeviceONU{
		NetworkDeviceID: device.ID,
		PONNo:           1,
		ONUNo:           8,
		Description:     "Optical Test",
		OperStatus:      "UP",
	}

	if err := db.Create(&onu).Error; err != nil {
		t.Fatal(err)
	}

	temperature := 39.5
	voltage := 3.21
	txBias := 15.2
	txPower := 2.15
	rxPower := -18.25

	opticalTime := now.Add(-10 * time.Minute)

	opticalSample := models.NetworkDeviceONUSample{
		NetworkDeviceONUID: onu.ID,
		SampledAt:          opticalTime,
		TemperatureC:       &temperature,
		VoltageV:           &voltage,
		TxBiasMA:           &txBias,
		TxPowerDBM:         &txPower,
		RxPowerDBM:         &rxPower,
	}

	latestTraffic := models.NetworkDeviceONUSample{
		NetworkDeviceONUID: onu.ID,
		SampledAt:          now,
		InMbps:             22.5,
		OutMbps:            6.75,
	}

	if err := db.Create(&opticalSample).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&latestTraffic).Error; err != nil {
		t.Fatal(err)
	}

	router := newNetworkDeviceONUTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices/"+itoaUint(device.ID)+"/onus",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status=%d body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response struct {
		ONUs []struct {
			LatestSample *struct {
				InMbps  float64 `json:"in_mbps"`
				OutMbps float64 `json:"out_mbps"`
			} `json:"latest_sample"`

			LatestOptical *struct {
				SampledAt    time.Time `json:"sampled_at"`
				TemperatureC *float64  `json:"temperature_c"`
				VoltageV     *float64  `json:"voltage_v"`
				TxBiasMA     *float64  `json:"tx_bias_ma"`
				TxPowerDBM   *float64  `json:"tx_power_dbm"`
				RxPowerDBM   *float64  `json:"rx_power_dbm"`
			} `json:"latest_optical"`
		} `json:"onus"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if len(response.ONUs) != 1 {
		t.Fatalf(
			"ONU count=%d want=1",
			len(response.ONUs),
		)
	}

	got := response.ONUs[0]

	if got.LatestSample == nil {
		t.Fatal("latest_sample is nil")
	}

	if got.LatestSample.InMbps != 22.5 ||
		got.LatestSample.OutMbps != 6.75 {
		t.Fatalf(
			"unexpected latest traffic: %+v",
			got.LatestSample,
		)
	}

	if got.LatestOptical == nil {
		t.Fatal("latest_optical is nil")
	}

	if !got.LatestOptical.SampledAt.Equal(opticalTime) {
		t.Fatalf(
			"optical sampled_at=%s want=%s",
			got.LatestOptical.SampledAt,
			opticalTime,
		)
	}

	if got.LatestOptical.RxPowerDBM == nil ||
		*got.LatestOptical.RxPowerDBM != rxPower {
		t.Fatalf(
			"rx_power_dbm=%v want=%v",
			got.LatestOptical.RxPowerDBM,
			rxPower,
		)
	}

	if got.LatestOptical.TxPowerDBM == nil ||
		*got.LatestOptical.TxPowerDBM != txPower {
		t.Fatalf(
			"tx_power_dbm=%v want=%v",
			got.LatestOptical.TxPowerDBM,
			txPower,
		)
	}
}

func TestListNetworkDeviceONUsHandlerEmpty(
	t *testing.T,
) {
	db := setupNetworkDeviceONUHandlerTestDB(t)
	device := createNetworkDeviceONUHandlerTestDevice(
		t,
		db,
	)

	router := newNetworkDeviceONUTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices/"+itoaUint(device.ID)+"/onus",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body = %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response struct {
		ONUs []json.RawMessage `json:"onus"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if len(response.ONUs) != 0 {
		t.Fatalf(
			"ONU count = %d, want 0",
			len(response.ONUs),
		)
	}
}

func TestListNetworkDeviceONUsHandlerInvalidID(
	t *testing.T,
) {
	setupNetworkDeviceONUHandlerTestDB(t)

	router := newNetworkDeviceONUTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices/not-a-number/onus",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusBadRequest,
		)
	}
}

func TestListNetworkDeviceONUsHandlerMissingDevice(
	t *testing.T,
) {
	setupNetworkDeviceONUHandlerTestDB(t)

	router := newNetworkDeviceONUTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices/999999/onus",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"status = %d, body = %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func itoaUint(value uint) string {
	return strconv.FormatUint(
		uint64(value),
		10,
	)
}
