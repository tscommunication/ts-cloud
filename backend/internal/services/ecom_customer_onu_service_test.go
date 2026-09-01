package services

import (
	"context"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
	ecommonitor "github.com/tscommunication/ts-cloud/internal/monitoring/ecom"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func encryptedECOMTestDevice(
	t *testing.T,
) models.NetworkDevice {
	t.Helper()

	snmpSecret, err := security.EncryptSecret(
		"snmp-community",
		networkDeviceManagementCredentialTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	managementSecret, err := security.EncryptSecret(
		"management-password",
		networkDeviceManagementCredentialTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	return models.NetworkDevice{
		ManagementIP:              "192.0.2.10",
		ManagementPort:            8010,
		ManagementUsername:        "admin-test",
		ManagementSecretEncrypted: managementSecret,
		SNMPVersion:               "V2C",
		SNMPPort:                  161,
		SNMPSecretEncrypted:       snmpSecret,
		MonitoringProtocol:        "SNMP",
		DeviceType:                "OLT",
		Vendor:                    "ECOM",
	}
}

func TestResolveECOMCustomerONUUsesCPEMACAndExactPosition(
	t *testing.T,
) {
	device := encryptedECOMTestDevice(t)
	device.ID = 9

	const cpeMAC = "04:95:E6:58:8E:E8"

	learnedCalled := false
	httpCalled := false
	finderCalled := false

	result, err := resolveECOMCustomerONU(
		context.Background(),
		&device,
		cpeMAC,
		networkDeviceManagementCredentialTestKey,
		func(
			cfg snmpmonitor.V2CConfig,
			macAddress string,
		) (*snmpmonitor.ECOMLearnedMACResolution, error) {
			learnedCalled = true

			if macAddress != cpeMAC {
				t.Fatalf(
					"learned MAC input = %q, want CPE MAC",
					macAddress,
				)
			}
			if cfg.Host != device.ManagementIP {
				t.Fatalf("SNMP host = %q", cfg.Host)
			}
			if cfg.Community != "snmp-community" {
				t.Fatal("SNMP credential was not decrypted correctly")
			}

			return &snmpmonitor.ECOMLearnedMACResolution{
				MACAddress: cpeMAC,
				VLAN:       3501,
				MACType:    2,
				PortID:     17825793,
				Interface:  "epon 0/1/1",
				PONNo:      1,
			}, nil
		},
		func(
			ctx context.Context,
			gotDevice *models.NetworkDevice,
			password string,
			macAddress string,
			evidence ecommonitor.LearnedMACEvidence,
		) (*ecommonitor.ExactONUResolution, error) {
			httpCalled = true

			if ctx == nil {
				t.Fatal("HTTP resolver context is nil")
			}
			if gotDevice.ID != 9 {
				t.Fatalf("HTTP device ID = %d, want 9", gotDevice.ID)
			}
			if password != "management-password" {
				t.Fatal(
					"management credential was not decrypted correctly",
				)
			}
			if macAddress != cpeMAC {
				t.Fatalf(
					"HTTP MAC input = %q, want CPE MAC",
					macAddress,
				)
			}
			if evidence.PortID != 17825793 ||
				evidence.Interface != "epon 0/1/1" ||
				evidence.PONNo != 1 ||
				evidence.VLAN != 3501 {
				t.Fatalf(
					"unexpected learned evidence: %#v",
					evidence,
				)
			}

			return &ecommonitor.ExactONUResolution{
				MACAddress: cpeMAC,
				Interface:  "epon 0/1/1",
				PONNo:      1,
				ONUNo:      1,
				VLAN:       3501,
				MACType:    1,
			}, nil
		},
		func(
			deviceID uint,
			ponNo int,
			onuNo int,
		) (*models.NetworkDeviceONU, error) {
			finderCalled = true

			if deviceID != 9 || ponNo != 1 || onuNo != 1 {
				t.Fatalf(
					"inventory lookup = device %d PON %d ONU %d",
					deviceID,
					ponNo,
					onuNo,
				)
			}

			return &models.NetworkDeviceONU{
				NetworkDeviceID: 9,
				PONNo:           1,
				ONUNo:           1,
				MACAddress:      "70:A5:6A:0C:37:A2",
				OperStatus:      "UP",
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !learnedCalled || !httpCalled || !finderCalled {
		t.Fatalf(
			"resolver calls: learned=%v http=%v finder=%v",
			learnedCalled,
			httpCalled,
			finderCalled,
		)
	}
	if result == nil || result.ONU == nil {
		t.Fatal("expected correlated ONU inventory result")
	}

	if result.ONU.MACAddress != "70:A5:6A:0C:37:A2" {
		t.Fatalf(
			"ONU inventory MAC = %q, want ONU MAC",
			result.ONU.MACAddress,
		)
	}
	if result.LearnedMAC.MACAddress != cpeMAC {
		t.Fatalf(
			"learned MAC = %q, want CPE MAC",
			result.LearnedMAC.MACAddress,
		)
	}
}

func TestResolveECOMCustomerONUNeverUsesCPEMACAsONUIdentity(
	t *testing.T,
) {
	device := encryptedECOMTestDevice(t)
	device.ID = 9

	const cpeMAC = "04:95:E6:58:8E:E8"

	result, err := resolveECOMCustomerONU(
		context.Background(),
		&device,
		cpeMAC,
		networkDeviceManagementCredentialTestKey,
		func(
			_ snmpmonitor.V2CConfig,
			_ string,
		) (*snmpmonitor.ECOMLearnedMACResolution, error) {
			return &snmpmonitor.ECOMLearnedMACResolution{
				MACAddress: cpeMAC,
				VLAN:       3501,
				PortID:     17825793,
				Interface:  "epon 0/1/1",
				PONNo:      1,
			}, nil
		},
		func(
			_ context.Context,
			_ *models.NetworkDevice,
			_ string,
			_ string,
			_ ecommonitor.LearnedMACEvidence,
		) (*ecommonitor.ExactONUResolution, error) {
			return &ecommonitor.ExactONUResolution{
				MACAddress: cpeMAC,
				Interface:  "epon 0/1/1",
				PONNo:      1,
				ONUNo:      1,
				VLAN:       3501,
			}, nil
		},
		func(
			_ uint,
			_ int,
			_ int,
		) (*models.NetworkDeviceONU, error) {
			return &models.NetworkDeviceONU{
				NetworkDeviceID: 9,
				PONNo:           1,
				ONUNo:           1,
				MACAddress:      "70:A5:6A:0C:37:A2",
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.ONU == nil {
		t.Fatal("expected result")
	}

	if result.ONU.MACAddress == cpeMAC {
		t.Fatal("CPE MAC must never become ONU inventory identity")
	}
}

func TestResolveECOMCustomerONUStopsWhenLearnedMACMissing(
	t *testing.T,
) {
	device := encryptedECOMTestDevice(t)
	device.ID = 9

	httpCalled := false
	finderCalled := false

	result, err := resolveECOMCustomerONU(
		context.Background(),
		&device,
		"04:95:E6:58:8E:E8",
		networkDeviceManagementCredentialTestKey,
		func(
			_ snmpmonitor.V2CConfig,
			_ string,
		) (*snmpmonitor.ECOMLearnedMACResolution, error) {
			return nil, nil
		},
		func(
			_ context.Context,
			_ *models.NetworkDevice,
			_ string,
			_ string,
			_ ecommonitor.LearnedMACEvidence,
		) (*ecommonitor.ExactONUResolution, error) {
			httpCalled = true
			return nil, nil
		},
		func(
			_ uint,
			_ int,
			_ int,
		) (*models.NetworkDeviceONU, error) {
			finderCalled = true
			return nil, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Fatalf("result = %#v, want nil", result)
	}
	if httpCalled || finderCalled {
		t.Fatalf(
			"downstream calls: http=%v finder=%v, want false",
			httpCalled,
			finderCalled,
		)
	}
}
