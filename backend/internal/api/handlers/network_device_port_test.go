package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupNetworkDevicePortHandlerTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			filepath.Join(
				t.TempDir(),
				"network-device-port-handler.db",
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
		&models.AgentNetworkDevice{},
		&models.NetworkDevicePort{},
		&models.NetworkDevicePortSample{},
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

func newNetworkDevicePortTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.GET(
		"/network/devices/:id/ports",
		ListNetworkDevicePorts,
	)

	return router
}

func TestListNetworkDevicePortsHandler(
	t *testing.T,
) {
	db := setupNetworkDevicePortHandlerTestDB(t)

	device := models.NetworkDevice{
		Code:               "OLT-TEST",
		Name:               "Test OLT",
		DeviceType:         "OLT",
		Vendor:             "VSOL",
		DeviceModel:        "V1600D",
		OLTType:            "EPON",
		ManagementIP:       "192.0.2.10",
		ManagementPort:     9001,
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		PollingInterval:    300,
		MonitoringEnabled:  true,
		MonitoringStatus:   "ONLINE",
	}

	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	ifIndex := 4
	now := time.Date(
		2026,
		time.August,
		23,
		6,
		30,
		0,
		0,
		time.UTC,
	)

	port := models.NetworkDevicePort{
		NetworkDeviceID: device.ID,
		PortKey:         "ifindex:4",
		IfIndex:         &ifIndex,
		Name:            "GE0/4",
		Description:     "Uplink",
		PortType:        "ETHERNET",
		AdminStatus:     "UP",
		OperStatus:      "UP",
		SpeedMbps:       1000,
		MACAddress:      "AA:BB:CC:DD:EE:FF",
		LastSeenAt:      &now,
	}

	if err := db.Create(&port).Error; err != nil {
		t.Fatal(err)
	}

	sample := models.NetworkDevicePortSample{
		NetworkDevicePortID: port.ID,
		SampledAt:           now,
		InOctets:            1000,
		OutOctets:           2000,
		InMbps:              12.5,
		OutMbps:             25.5,
		InErrors:            1,
		OutErrors:           2,
		InDiscards:          3,
		OutDiscards:         4,
	}

	if err := db.Create(&sample).Error; err != nil {
		t.Fatal(err)
	}

	router := newNetworkDevicePortTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/network/devices/%d/ports",
			device.ID,
		),
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
		Ports []struct {
			ID           uint   `json:"id"`
			Name         string `json:"name"`
			OperStatus   string `json:"oper_status"`
			SpeedMbps    int64  `json:"speed_mbps"`
			LatestSample *struct {
				InMbps  float64 `json:"in_mbps"`
				OutMbps float64 `json:"out_mbps"`
			} `json:"latest_sample"`
		} `json:"ports"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if len(response.Ports) != 1 {
		t.Fatalf(
			"port count = %d, want 1",
			len(response.Ports),
		)
	}

	got := response.Ports[0]

	if got.Name != "GE0/4" ||
		got.OperStatus != "UP" ||
		got.SpeedMbps != 1000 {
		t.Fatalf(
			"unexpected port response: %+v",
			got,
		)
	}

	if got.LatestSample == nil {
		t.Fatal("expected latest sample")
	}

	if got.LatestSample.InMbps != 12.5 ||
		got.LatestSample.OutMbps != 25.5 {
		t.Fatalf(
			"unexpected latest rates: %+v",
			got.LatestSample,
		)
	}
}

func TestListNetworkDevicePortsHandlerInvalidID(
	t *testing.T,
) {
	router := newNetworkDevicePortTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices/not-a-number/ports",
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

func TestListNetworkDevicePortsHandlerMissingDevice(
	t *testing.T,
) {
	setupNetworkDevicePortHandlerTestDB(t)

	router := newNetworkDevicePortTestRouter()
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices/999999/ports",
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
