package snmp

import (
	"math"
	"testing"
	"time"
)

func TestResolveONUVendorAdapterVSOLByVendor(
	t *testing.T,
) {
	adapter, ok := ResolveONUVendorAdapter(
		"vsol",
		"",
	)

	if !ok {
		t.Fatal("expected VSOL adapter")
	}

	if adapter.Name() != "VSOL" {
		t.Fatalf(
			"unexpected adapter %q",
			adapter.Name(),
		)
	}
}

func TestResolveONUVendorAdapterVSOLBySysObjectID(
	t *testing.T,
) {
	adapter, ok := ResolveONUVendorAdapter(
		"",
		".1.3.6.1.4.1.37950.1.1.5.10.14.1",
	)

	if !ok {
		t.Fatal("expected VSOL adapter")
	}

	if adapter.Name() != "VSOL" {
		t.Fatalf(
			"unexpected adapter %q",
			adapter.Name(),
		)
	}
}

func TestResolveONUVendorAdapterUnsupported(
	t *testing.T,
) {
	if _, ok := ResolveONUVendorAdapter(
		"UNKNOWN",
		".1.3.6.1.4.1.99999.1",
	); ok {
		t.Fatal(
			"unexpected adapter for unsupported vendor",
		)
	}
}

func TestParseVSOLONUOpticalRows(
	t *testing.T,
) {
	root := VSOLONUOpticalRootOID

	rows := []WalkResult{
		{
			OID:   root + ".1.1.2",
			Value: 1,
		},
		{
			OID:   root + ".2.1.2",
			Value: 2,
		},
		{
			OID:   root + ".3.1.2",
			Value: "35.00 C",
		},
		{
			OID:   root + ".4.1.2",
			Value: []byte("3.22 V"),
		},
		{
			OID:   root + ".5.1.2",
			Value: "15.00 mA",
		},
		{
			OID:   root + ".6.1.2",
			Value: "1.67 mW (2.22 dBm)",
		},
		{
			OID:   root + ".7.1.2",
			Value: "0.05 mW (-12.64 dBm)",
		},
	}

	got, err := ParseVSOLONUOpticalRows(rows)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"expected 1 ONU, got %d",
			len(got),
		)
	}

	onu := got[0]

	if onu.PONNo != 1 ||
		onu.ONUNo != 2 {
		t.Fatalf(
			"unexpected ONU key PON=%d ONU=%d",
			onu.PONNo,
			onu.ONUNo,
		)
	}

	assertFloatPointer(
		t,
		"temperature",
		onu.TemperatureC,
		35.00,
	)

	assertFloatPointer(
		t,
		"voltage",
		onu.VoltageV,
		3.22,
	)

	assertFloatPointer(
		t,
		"bias",
		onu.TxBiasMA,
		15.00,
	)

	assertFloatPointer(
		t,
		"tx power",
		onu.TxPowerDBM,
		2.22,
	)

	assertFloatPointer(
		t,
		"rx power",
		onu.RxPowerDBM,
		-12.64,
	)
}

func TestParseVSOLONUOpticalRowsAllowsOfflineBlankOptics(
	t *testing.T,
) {
	root := VSOLONUOpticalRootOID

	rows := []WalkResult{
		{
			OID:   root + ".1.3.10",
			Value: 3,
		},
		{
			OID:   root + ".2.3.10",
			Value: 10,
		},
		{
			OID:   root + ".3.3.10",
			Value: "",
		},
		{
			OID:   root + ".4.3.10",
			Value: "",
		},
		{
			OID:   root + ".5.3.10",
			Value: "",
		},
		{
			OID:   root + ".6.3.10",
			Value: "",
		},
		{
			OID:   root + ".7.3.10",
			Value: "",
		},
	}

	got, err := ParseVSOLONUOpticalRows(rows)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"expected 1 ONU, got %d",
			len(got),
		)
	}

	onu := got[0]

	if onu.PONNo != 3 ||
		onu.ONUNo != 10 {
		t.Fatal("unexpected ONU key")
	}

	if onu.TemperatureC != nil ||
		onu.VoltageV != nil ||
		onu.TxBiasMA != nil ||
		onu.TxPowerDBM != nil ||
		onu.RxPowerDBM != nil {
		t.Fatal(
			"blank optical fields must remain nil",
		)
	}
}

func assertFloatPointer(
	t *testing.T,
	name string,
	got *float64,
	want float64,
) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s is nil", name)
	}

	if math.Abs(*got-want) > 0.000001 {
		t.Fatalf(
			"%s = %f, want %f",
			name,
			*got,
			want,
		)
	}
}

func TestVSOLONUAdapterBuildPersistenceCandidates(
	t *testing.T,
) {
	sampledAt := time.Date(
		2026,
		time.August,
		23,
		7,
		40,
		0,
		0,
		time.UTC,
	)

	in := uint64(1000)
	out := uint64(2000)
	rx := -14.50

	ifmib := &IFMIBCollection{
		SampledAt: sampledAt,
		Ports: []IFMIBPort{
			{
				IfIndex:     15,
				Name:        "EPON01ONU3",
				OperStatus:  1,
				HCInOctets:  &in,
				HCOutOctets: &out,
			},
		},
	}

	optical := &ONUOpticalCollection{
		Vendor:    "VSOL",
		SampledAt: sampledAt,
		Records: []ONUOpticalRecord{
			{
				PONNo:      1,
				ONUNo:      3,
				RxPowerDBM: &rx,
			},
		},
	}

	adapter := VSOLONUAdapter{}

	got, err := adapter.BuildPersistenceCandidates(
		ifmib,
		optical,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"candidate count=%d want=1",
			len(got),
		)
	}

	if got[0].PONNo != 1 ||
		got[0].ONUNo != 3 ||
		got[0].IfIndex != 15 ||
		got[0].OperStatus != "UP" {
		t.Fatalf(
			"unexpected candidate PON=%d ONU=%d ifIndex=%d status=%q",
			got[0].PONNo,
			got[0].ONUNo,
			got[0].IfIndex,
			got[0].OperStatus,
		)
	}

	if got[0].RxPowerDBM == nil ||
		*got[0].RxPowerDBM != rx {
		t.Fatal(
			"unexpected correlated RX optical power",
		)
	}
}
