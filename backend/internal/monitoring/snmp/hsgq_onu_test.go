package snmp

import "testing"

func TestHSGQONUAdapterMatches(t *testing.T) {
	adapter := HSGQONUAdapter{}

	if !adapter.Matches("HSGQ", "") {
		t.Fatal("expected HSGQ vendor match")
	}

	if !adapter.Matches("", ".1.3.6.1.4.1.50224.3.1.1") {
		t.Fatal("expected HSGQ enterprise OID match")
	}

	if adapter.Matches("VSOL", ".1.3.6.1.4.1.37950") {
		t.Fatal("unexpected VSOL match")
	}
}

func TestParseHSGQONUDeviceIndex(t *testing.T) {
	ponNo, onuNo, ok := parseHSGQONUDeviceIndex(16777473)

	if !ok || ponNo != 1 || onuNo != 1 {
		t.Fatalf(
			"unexpected device index parse: pon=%d onu=%d ok=%v",
			ponNo,
			onuNo,
			ok,
		)
	}

	if _, _, ok := parseHSGQONUDeviceIndex(16777472); ok {
		t.Fatal("expected OLT-level index rejection")
	}
}

func TestParseHSGQONUOpticalRows(t *testing.T) {
	got, err := ParseHSGQONUOpticalRows(
		[]WalkResult{
			{
				OID: HSGQONUOpticalRxPowerOID +
					".16777472.65535.65535",
				Value: -2147483648,
			},
			{
				OID: HSGQONUOpticalRxPowerOID +
					".16777473.0.0",
				Value: -1405,
			},
			{
				OID: HSGQONUOpticalRxPowerOID +
					".16777474.0.0",
				Value: -890,
			},
		},
		[]WalkResult{
			{
				OID: HSGQONUOpticalTxPowerOID +
					".16777473.0.0",
				Value: 226,
			},
			{
				OID: HSGQONUOpticalTxPowerOID +
					".16777474.0.0",
				Value: 215,
			},
		},
	)
	if err != nil {
		t.Fatalf("parse optical rows: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}

	if got[0].IfIndex != 16777473 ||
		got[0].RxPowerDBM == nil ||
		*got[0].RxPowerDBM != -14.05 ||
		got[0].TxPowerDBM == nil ||
		*got[0].TxPowerDBM != 2.26 {
		t.Fatalf("unexpected first record: %+v", got[0])
	}
}

func TestParseHSGQONUOpticalOIDRejectsNonONUEntry(t *testing.T) {
	if _, ok := parseHSGQONUOpticalOID(
		HSGQONUOpticalRxPowerOID+".16777473.1.1",
		HSGQONUOpticalRxPowerOID,
	); ok {
		t.Fatal("expected non-ONU optical suffix rejection")
	}
}

func TestParseHSGQCentiDBMPointerRejectsSentinel(t *testing.T) {
	if value := parseHSGQCentiDBMPointer("-2147483648"); value != nil {
		t.Fatalf("expected nil sentinel value, got %v", *value)
	}
}

func TestBuildHSGQONUInventoryCandidates(t *testing.T) {
	inventory := &HSGQONUInventoryCollection{
		Vendor: "HSGQ",
		Records: []HSGQONUInventoryRecord{
			{
				Index:       16777474,
				PONNo:       1,
				ONUNo:       2,
				Description: "ONU01/02",
				MACAddress:  "04:F0:E4:01:7A:50",
				OperStatus:  "DOWN",
				Model:       "G55",
			},
			{
				Index:       16777473,
				PONNo:       1,
				ONUNo:       1,
				Description: "ONU01/01",
				MACAddress:  "04:F0:E4:01:7A:4F",
				OperStatus:  "UP",
				Model:       "G55",
			},
		},
	}

	got := BuildHSGQONUInventoryCandidates(inventory)

	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}

	if got[0].PONNo != 1 ||
		got[0].ONUNo != 1 ||
		got[0].IfIndex != 16777473 ||
		got[0].OperStatus != "UP" ||
		got[0].Model != "G55" {
		t.Fatalf("unexpected first candidate: %+v", got[0])
	}
}

func TestMergeHSGQONUOptical(t *testing.T) {
	rxPower := -14.05
	txPower := 2.26

	got := MergeHSGQONUOptical(
		[]ONUPersistenceCandidate{
			{PONNo: 1, ONUNo: 1, IfIndex: 16777473},
			{PONNo: 1, ONUNo: 2, IfIndex: 16777474},
		},
		&ONUOpticalCollection{
			Vendor: "HSGQ",
			Records: []ONUOpticalRecord{
				{
					IfIndex:    16777473,
					RxPowerDBM: &rxPower,
					TxPowerDBM: &txPower,
				},
			},
		},
	)

	if got[0].RxPowerDBM == nil ||
		*got[0].RxPowerDBM != -14.05 ||
		got[0].TxPowerDBM == nil ||
		*got[0].TxPowerDBM != 2.26 {
		t.Fatalf("expected matching optical values: %+v", got[0])
	}

	if got[1].RxPowerDBM != nil || got[1].TxPowerDBM != nil {
		t.Fatalf("unexpected unmatched optical values: %+v", got[1])
	}
}
