package snmp

import (
	"testing"
	"time"
)

func TestIFIndexFromOID(t *testing.T) {
	index, err := IFIndexFromOID(
		IFNameOID,
		".1.3.6.1.2.1.31.1.1.1.1.17825794",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if index != 17825794 {
		t.Fatalf("expected 17825794, got %d", index)
	}
}

func TestIFIndexFromOIDRejectsWrongSubtree(t *testing.T) {
	_, err := IFIndexFromOID(
		IFNameOID,
		".1.3.6.1.2.1.2.2.1.2.1",
	)
	if err == nil {
		t.Fatal("expected subtree validation error")
	}
}

func TestStringValue(t *testing.T) {
	value, err := StringValue([]byte("eth0/0/1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "eth0/0/1" {
		t.Fatalf("unexpected value %q", value)
	}
}

func TestUint64ValueCounter64(t *testing.T) {
	value, err := Uint64Value(
		uint64(13648076113270),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != 13648076113270 {
		t.Fatalf("unexpected value %d", value)
	}
}

func TestMACAddressValue(t *testing.T) {
	value, err := MACAddressValue(
		[]byte{0xf4, 0x70, 0x0c, 0x87, 0x2a, 0x5b},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "F4:70:0C:87:2A:5B" {
		t.Fatalf("unexpected MAC %q", value)
	}
}

func TestEffectiveSpeedMbpsPrefersHighSpeed(t *testing.T) {
	got := EffectiveSpeedMbps(
		10000,
		4294967295,
	)

	if got != 10000 {
		t.Fatalf("expected 10000 Mbps, got %d", got)
	}
}

func TestEffectiveSpeedMbpsFallsBackToIfSpeed(t *testing.T) {
	got := EffectiveSpeedMbps(
		0,
		1000000000,
	)

	if got != 1000 {
		t.Fatalf("expected 1000 Mbps, got %d", got)
	}
}

func TestEffectiveSpeedMbpsUnknown(t *testing.T) {
	got := EffectiveSpeedMbps(0, 0)

	if got != 0 {
		t.Fatalf("expected unknown speed 0, got %d", got)
	}
}

func TestMergeIFMIBColumnsBuildsPort(t *testing.T) {
	ports, err := MergeIFMIBColumns(
		[]IFMIBColumn{
			{
				RootOID: IFNameOID,
				Rows: []WalkResult{
					{
						OID:   IFNameOID + ".1",
						Value: []byte("eth0/0/1"),
					},
				},
			},
			{
				RootOID: IFDescrOID,
				Rows: []WalkResult{
					{
						OID:   IFDescrOID + ".1",
						Value: []byte("uplink"),
					},
				},
			},
			{
				RootOID: IFHighSpeedOID,
				Rows: []WalkResult{
					{
						OID:   IFHighSpeedOID + ".1",
						Value: uint32(1000),
					},
				},
			},
			{
				RootOID: IFHCInOctetsOID,
				Rows: []WalkResult{
					{
						OID:   IFHCInOctetsOID + ".1",
						Value: uint64(12345),
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	port := ports[1]
	if port == nil {
		t.Fatal("expected port 1")
	}

	if port.Name != "eth0/0/1" {
		t.Fatalf("unexpected name %q", port.Name)
	}

	if port.Description != "uplink" {
		t.Fatalf(
			"unexpected description %q",
			port.Description,
		)
	}

	if port.HighSpeedMbps != 1000 {
		t.Fatalf(
			"unexpected high speed %d",
			port.HighSpeedMbps,
		)
	}

	if port.HCInOctets == nil ||
		*port.HCInOctets != 12345 {
		t.Fatal("unexpected HC in octets")
	}
}

func TestMergeIFMIBColumnsAllowsMissingOptionalCounters(
	t *testing.T,
) {
	ports, err := MergeIFMIBColumns(
		[]IFMIBColumn{
			{
				RootOID: IFNameOID,
				Rows: []WalkResult{
					{
						OID:   IFNameOID + ".8",
						Value: []byte("eth0/0/8"),
					},
				},
			},
			{
				RootOID: IFInErrorsOID,
				Rows: []WalkResult{
					{
						OID:   IFInErrorsOID + ".8",
						Value: uint32(0),
					},
				},
			},
			{
				RootOID: IFInDiscardsOID,
				Rows: []WalkResult{
					{
						OID:   IFInDiscardsOID + ".8",
						Value: uint32(11011),
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	port := ports[8]
	if port == nil {
		t.Fatal("expected port 8")
	}

	if port.OutErrors != nil {
		t.Fatal("expected missing out errors to remain nil")
	}

	if port.OutDiscards != nil {
		t.Fatal("expected missing out discards to remain nil")
	}

	if port.InDiscards == nil ||
		*port.InDiscards != 11011 {
		t.Fatal("unexpected in discards")
	}
}

func TestMergeIFMIBColumnsHandlesLargeIfIndex(t *testing.T) {
	ports, err := MergeIFMIBColumns(
		[]IFMIBColumn{
			{
				RootOID: IFNameOID,
				Rows: []WalkResult{
					{
						OID: IFNameOID +
							".17825794",
						Value: []byte("VLAN-IF20"),
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	port := ports[17825794]
	if port == nil {
		t.Fatal("expected VLAN interface")
	}

	if port.Name != "VLAN-IF20" {
		t.Fatalf("unexpected name %q", port.Name)
	}
}

func TestInterfaceStatus(t *testing.T) {
	tests := []struct {
		value int
		want  string
	}{
		{1, "UP"},
		{2, "DOWN"},
		{3, "TESTING"},
		{4, "UNKNOWN"},
		{5, "DORMANT"},
		{6, "NOT_PRESENT"},
		{7, "LOWER_LAYER_DOWN"},
		{99, "UNKNOWN"},
	}

	for _, test := range tests {
		got := InterfaceStatus(test.value)

		if got != test.want {
			t.Fatalf(
				"value=%d expected=%q got=%q",
				test.value,
				test.want,
				got,
			)
		}
	}
}

func TestLastChangeAt(t *testing.T) {
	sampledAt := time.Date(
		2026,
		8,
		23,
		5,
		30,
		0,
		0,
		time.UTC,
	)

	got := LastChangeAt(
		sampledAt,
		10000,
		7000,
	)

	if got == nil {
		t.Fatal("expected timestamp")
	}

	want := sampledAt.Add(
		-30 * time.Second,
	)

	if !got.Equal(want) {
		t.Fatalf(
			"expected %s got %s",
			want,
			got,
		)
	}
}

func TestLastChangeAtZeroReturnsNil(t *testing.T) {
	sampledAt := time.Now()

	if got := LastChangeAt(
		sampledAt,
		10000,
		0,
	); got != nil {
		t.Fatal("expected nil for zero lastChange")
	}
}

func TestLastChangeAtRejectsImpossibleTicks(t *testing.T) {
	sampledAt := time.Now()

	if got := LastChangeAt(
		sampledAt,
		100,
		200,
	); got != nil {
		t.Fatal("expected nil for lastChange > uptime")
	}
}
