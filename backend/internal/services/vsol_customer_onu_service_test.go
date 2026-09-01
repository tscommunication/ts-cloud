package services

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/security"
)

const vsolCustomerONUTestKey = "0123456789abcdef0123456789abcdef"

func testVSOLDevice(
	t *testing.T,
	vendor string,
) *models.NetworkDevice {
	t.Helper()

	encrypted, err := security.EncryptSecret(
		"public-test-community",
		vsolCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	return &models.NetworkDevice{
		DeviceType:          "OLT",
		Vendor:              vendor,
		ManagementIP:        "192.0.2.10",
		MonitoringProtocol:  "SNMP",
		SNMPVersion:         "V2C",
		SNMPPort:            161,
		SNMPSecretEncrypted: encrypted,
	}
}

func TestResolveVSOLCustomerONUFindsInventory(t *testing.T) {
	device := testVSOLDevice(t, "VSOL")
	device.ID = 5

	learnedResolver := func(
		cfg snmpmonitor.V2CConfig,
		macAddress string,
	) (*snmpmonitor.VSOLLearnedMACResolution, error) {
		if cfg.Host != "192.0.2.10" {
			t.Fatalf("host=%q", cfg.Host)
		}

		if cfg.Community != "public-test-community" {
			t.Fatal("unexpected decrypted community")
		}

		if macAddress != "58:D9:D5:27:3E:F7" {
			t.Fatalf("MAC=%q", macAddress)
		}

		return &snmpmonitor.VSOLLearnedMACResolution{
			MACAddress: macAddress,
			PortID:     19,
			Interface:  "EPON0/1:4",
			PONNo:      1,
			ONUNo:      4,
		}, nil
	}

	onuFinder := func(
		deviceID uint,
		ponNo int,
		onuNo int,
	) (*models.NetworkDeviceONU, error) {
		if deviceID != 5 || ponNo != 1 || onuNo != 4 {
			t.Fatalf(
				"unexpected position device=%d PON=%d ONU=%d",
				deviceID,
				ponNo,
				onuNo,
			)
		}

		ifIndex := 19

		return &models.NetworkDeviceONU{
			NetworkDeviceID: 5,
			PONNo:           1,
			ONUNo:           4,
			IfIndex:         &ifIndex,
		}, nil
	}

	got, err := resolveVSOLCustomerONU(
		device,
		"58:D9:D5:27:3E:F7",
		vsolCustomerONUTestKey,
		learnedResolver,
		onuFinder,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil || got.ONU == nil || got.LearnedMAC == nil {
		t.Fatalf("unexpected nil resolution: %+v", got)
	}

	if got.ONU.PONNo != 1 ||
		got.ONU.ONUNo != 4 ||
		got.ONU.IfIndex == nil ||
		*got.ONU.IfIndex != 19 {
		t.Fatalf("unexpected ONU: %+v", got.ONU)
	}
}

func TestResolveVSOLCustomerONUSupportsZIBBIX(t *testing.T) {
	device := testVSOLDevice(t, "ZIBBIX")
	device.ID = 1

	learnedResolver := func(
		cfg snmpmonitor.V2CConfig,
		macAddress string,
	) (*snmpmonitor.VSOLLearnedMACResolution, error) {
		return nil, nil
	}

	onuFinder := func(
		deviceID uint,
		ponNo int,
		onuNo int,
	) (*models.NetworkDeviceONU, error) {
		t.Fatal("ONU finder must not run without learned MAC")
		return nil, nil
	}

	got, err := resolveVSOLCustomerONU(
		device,
		"00:11:22:33:44:55",
		vsolCustomerONUTestKey,
		learnedResolver,
		onuFinder,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got != nil {
		t.Fatalf("expected nil resolution, got %+v", got)
	}
}
