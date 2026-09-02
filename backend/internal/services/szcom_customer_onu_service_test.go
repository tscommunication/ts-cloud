package services

import (
	"context"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	szcommonitor "github.com/tscommunication/ts-cloud/internal/monitoring/szcom"
	"github.com/tscommunication/ts-cloud/internal/security"
)

const szcomCustomerONUTestKey = "0123456789abcdef0123456789abcdef"

func TestResolveSZCOMCustomerONU(
	t *testing.T,
) {
	snmpSecret, err := security.EncryptSecret(
		"public-test-community",
		szcomCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	managementSecret, err := security.EncryptSecret(
		"management-password",
		szcomCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	device := &models.NetworkDevice{
		DeviceType:                "OLT",
		Vendor:                    "SZCOM-SOLITINE",
		ManagementIP:              "192.0.2.30",
		ManagementPort:            9007,
		ManagementUsername:        "admin",
		ManagementSecretEncrypted: managementSecret,
		MonitoringProtocol:        "SNMP",
		SNMPVersion:               "V2C",
		SNMPPort:                  161,
		SNMPSecretEncrypted:       snmpSecret,
	}
	device.ID = 8

	ponResolver := func(
		cfg snmpmonitor.V2CConfig,
		macAddress string,
	) (*snmpmonitor.SZCOMLearnedMACPONResolution, error) {
		if cfg.Host != "192.0.2.30" ||
			cfg.Community != "public-test-community" {
			t.Fatalf("unexpected SNMP config: %+v", cfg)
		}

		if macAddress != "40:EE:15:73:EE:C9" {
			t.Fatalf("unexpected MAC %q", macAddress)
		}

		return &snmpmonitor.SZCOMLearnedMACPONResolution{
			MACAddress: macAddress,
			PortID:     5002,
			IfIndex:    5002,
			Interface:  "pon1",
			PONNo:      1,
		}, nil
	}

	cliResolver := func(
		_ context.Context,
		host string,
		port int,
		username string,
		password string,
		ponNo int,
		onuNos []int,
		macAddress string,
	) (*szcommonitor.LearnedMACResolution, error) {
		if host != "192.0.2.30" ||
			port != szcommonitor.DefaultTelnetPort ||
			username != "admin" ||
			password != "management-password" {
			t.Fatal("unexpected management credentials/config")
		}

		if ponNo != 1 ||
			macAddress != "40:EE:15:73:EE:C9" {
			t.Fatal("unexpected correlation input")
		}

		found12 := false
		for _, n := range onuNos {
			if n == 12 {
				found12 = true
			}
		}
		if !found12 {
			t.Fatalf("ONU12 missing from candidates: %v", onuNos)
		}

		return &szcommonitor.LearnedMACResolution{
			MACAddress: macAddress,
			PONNo:      1,
			ONUNo:      12,
			ETHPort:    1,
		}, nil
	}

	onuList := func(
		deviceID uint,
	) ([]models.NetworkDeviceONU, error) {
		if deviceID != 8 {
			t.Fatalf("unexpected device ID %d", deviceID)
		}

		return []models.NetworkDeviceONU{
			{
				ID:              101,
				NetworkDeviceID: 8,
				PONNo:           1,
				ONUNo:           1,
			},
			{
				ID:              112,
				NetworkDeviceID: 8,
				PONNo:           1,
				ONUNo:           12,
				MACAddress:      "80:F7:A6:3A:EA:5B",
			},
			{
				ID:              201,
				NetworkDeviceID: 8,
				PONNo:           2,
				ONUNo:           1,
			},
		}, nil
	}

	got, err := resolveSZCOMCustomerONU(
		context.Background(),
		device,
		"40:EE:15:73:EE:C9",
		szcomCustomerONUTestKey,
		ponResolver,
		cliResolver,
		func(
			context.Context,
			string,
			int,
			string,
			string,
			map[int][]int,
			string,
		) (*szcommonitor.LearnedMACResolution, error) {
			t.Fatal("across-PON resolver must not run on FDB fast path")
			return nil, nil
		},
		onuList,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil ||
		got.ONU == nil ||
		got.LearnedPON == nil ||
		got.LearnedMAC == nil {
		t.Fatalf("unexpected nil resolution: %+v", got)
	}

	if got.ONU.PONNo != 1 ||
		got.ONU.ONUNo != 12 ||
		got.ONU.MACAddress != "80:F7:A6:3A:EA:5B" {
		t.Fatalf("unexpected ONU: %+v", got.ONU)
	}
}

func TestResolveSZCOMCustomerONUFDBMissFallback(
	t *testing.T,
) {
	snmpSecret, err := security.EncryptSecret(
		"public-test-community",
		szcomCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	managementSecret, err := security.EncryptSecret(
		"management-test-password",
		szcomCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	device := &models.NetworkDevice{
		Code:                      "OLT-007",
		Vendor:                    "SZCOM-SOLITINE",
		DeviceType:                "OLT",
		ManagementIP:              "192.0.2.30",
		ManagementUsername:        "admin",
		ManagementSecretEncrypted: managementSecret,
		MonitoringProtocol:        "SNMP",
		SNMPVersion:               "V2C",
		SNMPPort:                  161,
		SNMPSecretEncrypted:       snmpSecret,
	}
	device.ID = 8

	got, err := resolveSZCOMCustomerONU(
		context.Background(),
		device,
		"40:EE:15:73:EE:C9",
		szcomCustomerONUTestKey,
		func(
			snmpmonitor.V2CConfig,
			string,
		) (*snmpmonitor.SZCOMLearnedMACPONResolution, error) {
			return nil, nil
		},
		func(
			context.Context,
			string,
			int,
			string,
			string,
			int,
			[]int,
			string,
		) (*szcommonitor.LearnedMACResolution, error) {
			t.Fatal("single-PON resolver must not run after FDB miss")
			return nil, nil
		},
		func(
			_ context.Context,
			host string,
			port int,
			username string,
			password string,
			ponONUs map[int][]int,
			macAddress string,
		) (*szcommonitor.LearnedMACResolution, error) {
			if host != "192.0.2.30" ||
				port != szcommonitor.DefaultTelnetPort ||
				username != "admin" ||
				password != "management-test-password" ||
				macAddress != "40:EE:15:73:EE:C9" {
				t.Fatal("unexpected across-PON management arguments")
			}

			if len(ponONUs) != 2 ||
				len(ponONUs[1]) != 2 ||
				len(ponONUs[2]) != 1 {
				t.Fatalf(
					"unexpected PON candidates: %+v",
					ponONUs,
				)
			}

			return &szcommonitor.LearnedMACResolution{
				MACAddress: macAddress,
				PONNo:      1,
				ONUNo:      12,
				ETHPort:    1,
			}, nil
		},
		func(
			deviceID uint,
		) ([]models.NetworkDeviceONU, error) {
			if deviceID != 8 {
				t.Fatalf(
					"unexpected device ID %d",
					deviceID,
				)
			}

			return []models.NetworkDeviceONU{
				{
					ID:              101,
					NetworkDeviceID: 8,
					PONNo:           1,
					ONUNo:           1,
				},
				{
					ID:              112,
					NetworkDeviceID: 8,
					PONNo:           1,
					ONUNo:           12,
				},
				{
					ID:              201,
					NetworkDeviceID: 8,
					PONNo:           2,
					ONUNo:           1,
				},
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil ||
		got.ONU == nil ||
		got.LearnedMAC == nil {
		t.Fatalf(
			"unexpected nil fallback resolution: %+v",
			got,
		)
	}

	if got.LearnedPON != nil {
		t.Fatalf(
			"fallback must not invent learned PON: %+v",
			got.LearnedPON,
		)
	}

	if got.ONU.PONNo != 1 ||
		got.ONU.ONUNo != 12 ||
		got.LearnedMAC.PONNo != 1 ||
		got.LearnedMAC.ONUNo != 12 {
		t.Fatalf(
			"unexpected fallback resolution: %+v",
			got,
		)
	}

}

func TestResolveSZCOMCustomerONUFDBMissAmbiguousSoftMiss(
	t *testing.T,
) {
	snmpSecret, err := security.EncryptSecret(
		"public-test-community",
		szcomCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	managementSecret, err := security.EncryptSecret(
		"management-test-password",
		szcomCustomerONUTestKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	device := &models.NetworkDevice{
		Code:                      "OLT-007",
		Vendor:                    "SZCOM-SOLITINE",
		DeviceType:                "OLT",
		ManagementIP:              "192.0.2.30",
		ManagementUsername:        "admin",
		ManagementSecretEncrypted: managementSecret,
		MonitoringProtocol:        "SNMP",
		SNMPVersion:               "V2C",
		SNMPPort:                  161,
		SNMPSecretEncrypted:       snmpSecret,
	}
	device.ID = 8

	got, err := resolveSZCOMCustomerONU(
		context.Background(),
		device,
		"40:EE:15:73:EE:C9",
		szcomCustomerONUTestKey,
		func(
			snmpmonitor.V2CConfig,
			string,
		) (*snmpmonitor.SZCOMLearnedMACPONResolution, error) {
			return nil, nil
		},
		func(
			context.Context,
			string,
			int,
			string,
			string,
			int,
			[]int,
			string,
		) (*szcommonitor.LearnedMACResolution, error) {
			t.Fatal("single-PON resolver must not run after FDB miss")
			return nil, nil
		},
		func(
			context.Context,
			string,
			int,
			string,
			string,
			map[int][]int,
			string,
		) (*szcommonitor.LearnedMACResolution, error) {
			// The client-level tests verify that cross-PON ambiguity
			// produces this nil soft miss.
			return nil, nil
		},
		func(
			uint,
		) ([]models.NetworkDeviceONU, error) {
			return []models.NetworkDeviceONU{
				{
					ID:              101,
					NetworkDeviceID: 8,
					PONNo:           1,
					ONUNo:           1,
				},
				{
					ID:              201,
					NetworkDeviceID: 8,
					PONNo:           2,
					ONUNo:           1,
				},
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got != nil {
		t.Fatalf(
			"expected ambiguous fallback soft miss, got %+v",
			got,
		)
	}
}
