package repositories

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindNetworkDeviceONUByIdentityNormalizesMAC(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:onu_identity?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	if err := db.AutoMigrate(&models.NetworkDevice{}, &models.NetworkDeviceONU{}); err != nil { t.Fatal(err) }
	device := models.NetworkDevice{Code: "OLT-IDENTITY", Name: "OLT Identity", DeviceType: "OLT", Vendor: "TEST", DeviceModel: "Test", ManagementIP: "10.0.0.10", MonitoringProtocol: "SNMP"}
	if err := db.Create(&device).Error; err != nil { t.Fatal(err) }
	onu := models.NetworkDeviceONU{NetworkDeviceID: device.ID, PONNo: 2, ONUNo: 7, MACAddress: "AA:BB:CC:DD:EE:FF", SerialNumber: "ONU-ABC"}
	if err := db.Create(&onu).Error; err != nil { t.Fatal(err) }

	found, err := FindNetworkDeviceONUByIdentity("aa-bb-cc-dd-ee-ff")
	if err != nil || found == nil || found.ID != onu.ID || found.NetworkDevice == nil || found.NetworkDevice.Code != device.Code {
		t.Fatalf("unexpected ONU identity lookup: %#v, %v", found, err)
	}
}
