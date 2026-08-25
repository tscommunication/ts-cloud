package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupNetworkDeviceAgentScopeTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			filepath.Join(
				t.TempDir(),
				"network-device-agent-scope.db",
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

func createAgentScopeNetworkDevice(
	t *testing.T,
	db *gorm.DB,
	code string,
	ip string,
) models.NetworkDevice {
	t.Helper()

	row := models.NetworkDevice{
		Code:               code,
		Name:               code,
		DeviceType:         "OLT",
		Vendor:             "VSOL",
		DeviceModel:        "TEST",
		OLTType:            "EPON",
		ManagementIP:       ip,
		ManagementPort:     9001,
		MonitoringProtocol: "SNMP",
		SNMPVersion:        "V2C",
		SNMPPort:           161,
		PollingInterval:    300,
		MonitoringEnabled:  true,
		MonitoringStatus:   "ONLINE",
	}

	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	return row
}

func newAgentScopeRouter(
	role string,
	agentID uint,
) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("role", role)
		c.Set("agent_id", agentID)
		c.Next()
	})

	cfg := &config.Config{}

	router.GET(
		"/network/devices",
		ListNetworkDevices(cfg),
	)
	router.GET(
		"/network/devices/:id/ports",
		ListNetworkDevicePorts,
	)
	router.GET(
		"/network/devices/:id/onus",
		ListNetworkDeviceONUs,
	)

	return router
}

func TestAgentNetworkDeviceListIsScoped(
	t *testing.T,
) {
	db := setupNetworkDeviceAgentScopeTestDB(t)

	assigned := createAgentScopeNetworkDevice(
		t,
		db,
		"OLT-ASSIGNED",
		"192.0.2.101",
	)
	_ = createAgentScopeNetworkDevice(
		t,
		db,
		"OLT-UNASSIGNED",
		"192.0.2.102",
	)

	const agentID uint = 77

	if err := db.Create(
		&models.AgentNetworkDevice{
			AgentID:         agentID,
			NetworkDeviceID: assigned.ID,
		},
	).Error; err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/network/devices",
		nil,
	)

	newAgentScopeRouter(
		"agent",
		agentID,
	).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var body struct {
		Devices []struct {
			ID uint `json:"id"`
		} `json:"devices"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatal(err)
	}

	if len(body.Devices) != 1 {
		t.Fatalf(
			"expected 1 assigned device, got %d",
			len(body.Devices),
		)
	}

	if body.Devices[0].ID != assigned.ID {
		t.Fatalf(
			"expected assigned device %d, got %d",
			assigned.ID,
			body.Devices[0].ID,
		)
	}
}

func TestAgentNetworkDevicePortsAccess(
	t *testing.T,
) {
	db := setupNetworkDeviceAgentScopeTestDB(t)

	assigned := createAgentScopeNetworkDevice(
		t,
		db,
		"OLT-PORT-A",
		"192.0.2.111",
	)
	unassigned := createAgentScopeNetworkDevice(
		t,
		db,
		"OLT-PORT-B",
		"192.0.2.112",
	)

	const agentID uint = 88

	if err := db.Create(
		&models.AgentNetworkDevice{
			AgentID:         agentID,
			NetworkDeviceID: assigned.ID,
		},
	).Error; err != nil {
		t.Fatal(err)
	}

	router := newAgentScopeRouter(
		"agent",
		agentID,
	)

	for _, tc := range []struct {
		name       string
		deviceID   uint
		wantStatus int
	}{
		{
			name:       "assigned",
			deviceID:   assigned.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unassigned",
			deviceID:   unassigned.ID,
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf(
					"/network/devices/%d/ports",
					tc.deviceID,
				),
				nil,
			)

			router.ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatus {
				t.Fatalf(
					"expected %d, got %d: %s",
					tc.wantStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestAgentNetworkDeviceONUsAccess(
	t *testing.T,
) {
	db := setupNetworkDeviceAgentScopeTestDB(t)

	assigned := createAgentScopeNetworkDevice(
		t,
		db,
		"OLT-ONU-A",
		"192.0.2.121",
	)
	unassigned := createAgentScopeNetworkDevice(
		t,
		db,
		"OLT-ONU-B",
		"192.0.2.122",
	)

	const agentID uint = 99

	if err := db.Create(
		&models.AgentNetworkDevice{
			AgentID:         agentID,
			NetworkDeviceID: assigned.ID,
		},
	).Error; err != nil {
		t.Fatal(err)
	}

	router := newAgentScopeRouter(
		"agent",
		agentID,
	)

	for _, tc := range []struct {
		name       string
		deviceID   uint
		wantStatus int
	}{
		{
			name:       "assigned",
			deviceID:   assigned.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unassigned",
			deviceID:   unassigned.ID,
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf(
					"/network/devices/%d/onus",
					tc.deviceID,
				),
				nil,
			)

			router.ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatus {
				t.Fatalf(
					"expected %d, got %d: %s",
					tc.wantStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestUnlinkedAgentNetworkDeviceAccessDenied(
	t *testing.T,
) {
	_ = setupNetworkDeviceAgentScopeTestDB(t)

	router := newAgentScopeRouter(
		"agent",
		0,
	)

	for _, path := range []string{
		"/network/devices",
		"/network/devices/1/ports",
		"/network/devices/1/onus",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			path,
			nil,
		)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf(
				"path %s: expected 403, got %d: %s",
				path,
				recorder.Code,
				recorder.Body.String(),
			)
		}
	}
}
