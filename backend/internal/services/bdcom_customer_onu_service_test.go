package services

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/security"
)

const bdcomCustomerONUTestKey = "0123456789abcdef0123456789abcdef"

func testBDCOMDevice(
	t *testing.T,
) *models.NetworkDevice {
	t.Helper()

	encrypted, err := security.EncryptSecret(
		"public-test-community",
		bdcomCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	return &models.NetworkDevice{
		DeviceType:          "OLT",
		Vendor:              "BDCOM",
		ManagementIP:        "192.0.2.14",
		MonitoringProtocol:  "SNMP",
		SNMPVersion:         "V2C",
		SNMPPort:            161,
		SNMPSecretEncrypted: encrypted,
	}
}

func TestResolveBDCOMCustomerONUFindsInventory(
	t *testing.T,
) {
	device := testBDCOMDevice(t)
	device.ID = 4

	learnedResolver := func(
		cfg snmpmonitor.V2CConfig,
		macAddress string,
	) (*snmpmonitor.BDCOMLearnedMACResolution, error) {
		if cfg.Host != "192.0.2.14" {
			t.Fatalf("host=%q", cfg.Host)
		}

		if cfg.Community != "public-test-community" {
			t.Fatal("unexpected decrypted community")
		}

		if macAddress != "50:D2:F5:B8:DC:4E" {
			t.Fatalf("MAC=%q", macAddress)
		}

		return &snmpmonitor.BDCOMLearnedMACResolution{
			MACAddress: macAddress,
			VLAN:       624,
			PortID:     284,
			Interface:  "EPON0/4:8",
			PONNo:      4,
			ONUNo:      8,
		}, nil
	}

	onuFinder := func(
		deviceID uint,
		ponNo int,
		onuNo int,
	) (*models.NetworkDeviceONU, error) {
		if deviceID != 4 ||
			ponNo != 4 ||
			onuNo != 8 {
			t.Fatalf(
				"unexpected position device=%d PON=%d ONU=%d",
				deviceID,
				ponNo,
				onuNo,
			)
		}

		ifIndex := 284

		return &models.NetworkDeviceONU{
			NetworkDeviceID: 4,
			PONNo:           4,
			ONUNo:           8,
			IfIndex:         &ifIndex,
		}, nil
	}

	got, err := resolveBDCOMCustomerONU(
		device,
		"50:D2:F5:B8:DC:4E",
		bdcomCustomerONUTestKey,
		learnedResolver,
		onuFinder,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil ||
		got.ONU == nil ||
		got.LearnedMAC == nil {
		t.Fatalf(
			"unexpected nil resolution: %+v",
			got,
		)
	}

	if got.ONU.PONNo != 4 ||
		got.ONU.ONUNo != 8 ||
		got.ONU.IfIndex == nil ||
		*got.ONU.IfIndex != 284 {
		t.Fatalf(
			"unexpected ONU: %+v",
			got.ONU,
		)
	}
}
