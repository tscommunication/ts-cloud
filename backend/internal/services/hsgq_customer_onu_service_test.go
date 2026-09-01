package services

import (
	"context"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
	hsgqmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/hsgq"
	"github.com/tscommunication/ts-cloud/internal/security"
)

func TestResolveHSGQCustomerONU(t *testing.T) {
	const key = "0123456789abcdef0123456789abcdef"

	encrypted, err := security.EncryptSecret(
		"management-password",
		key,
	)
	if err != nil {
		t.Fatalf(
			"encrypt management secret: %v",
			err,
		)
	}

	device := &models.NetworkDevice{
		ManagementPort:            80,
		ManagementUsername:        "root",
		ManagementSecretEncrypted: encrypted,
	}
	device.ID = 6
	device.DeviceType = "OLT"
	device.Vendor = "HSGQ"
	device.ManagementIP = "192.0.2.1"

	expectedONU := &models.NetworkDeviceONU{}
	expectedONU.ID = 123

	got, err := resolveHSGQCustomerONU(
		context.Background(),
		device,
		"80:AF:CA:AE:6A:B5",
		key,
		func(
			ctx context.Context,
			gotDevice *models.NetworkDevice,
			password string,
			macAddress string,
		) (*hsgqmonitor.LearnedMACResolution, error) {
			if ctx == nil {
				t.Fatal("context is nil")
			}
			if gotDevice.ID != device.ID {
				t.Fatalf(
					"device=%d want=%d",
					gotDevice.ID,
					device.ID,
				)
			}
			if password != "management-password" {
				t.Fatalf(
					"unexpected decrypted password",
				)
			}
			if macAddress != "80:AF:CA:AE:6A:B5" {
				t.Fatalf(
					"MAC=%q",
					macAddress,
				)
			}

			return &hsgqmonitor.LearnedMACResolution{
				MACAddress: "80:af:ca:ae:6a:b5",
				VLANID:     1,
				PONNo:      1,
				ONUNo:      4,
				MACType:    0,
				ONUName:    "ONU01/04",
			}, nil
		},
		func(
			deviceID uint,
			ponNo int,
			onuNo int,
		) (*models.NetworkDeviceONU, error) {
			if deviceID != 6 ||
				ponNo != 1 ||
				onuNo != 4 {
				t.Fatalf(
					"unexpected lookup: device=%d pon=%d onu=%d",
					deviceID,
					ponNo,
					onuNo,
				)
			}

			return expectedONU, nil
		},
	)
	if err != nil {
		t.Fatalf(
			"resolve HSGQ customer ONU: %v",
			err,
		)
	}

	if got == nil ||
		got.ONU == nil ||
		got.ONU.ID != expectedONU.ID ||
		got.LearnedMAC == nil ||
		got.LearnedMAC.PONNo != 1 ||
		got.LearnedMAC.ONUNo != 4 {
		t.Fatalf(
			"unexpected resolution: %+v",
			got,
		)
	}
}
