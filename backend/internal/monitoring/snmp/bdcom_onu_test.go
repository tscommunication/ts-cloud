package snmp

import (
	"testing"
	"time"
)

func TestParseBDCOMONUIfDescr(t *testing.T) {
	pon, onu, ok := parseBDCOMONUIfDescr(
		"EPON0/3:14",
	)

	if !ok {
		t.Fatal("expected BDCOM ONU interface match")
	}

	if pon != 3 || onu != 14 {
		t.Fatalf(
			"PON=%d ONU=%d want PON=3 ONU=14",
			pon,
			onu,
		)
	}
}

func TestParseBDCOMONUIfDescrRejectsPONInterface(
	t *testing.T,
) {
	_, _, ok := parseBDCOMONUIfDescr(
		"EPON0/3",
	)

	if ok {
		t.Fatal(
			"PON interface must not be parsed as ONU",
		)
	}
}

func TestParseBDCOMONUStatus(t *testing.T) {
	tests := map[string]string{
		"3": "UP",
		"2": "DOWN",
		"1": "UNKNOWN",
		"":  "UNKNOWN",
	}

	for input, want := range tests {
		if got := parseBDCOMONUStatus(input); got != want {
			t.Fatalf(
				"status %q=%q want=%q",
				input,
				got,
				want,
			)
		}
	}
}

func TestBDCOMONUAdapterBuildPersistenceCandidates(
	t *testing.T,
) {
	sampledAt := time.Date(
		2026,
		time.August,
		26,
		12,
		15,
		0,
		0,
		time.UTC,
	)

	in := uint64(1234)
	out := uint64(5678)

	ifmib := &IFMIBCollection{
		SampledAt: sampledAt,
		Ports: []IFMIBPort{
			{
				IfIndex:     84,
				Name:        "EPON0/1:1",
				OperStatus:  1,
				HCInOctets:  &in,
				HCOutOctets: &out,
			},
			{
				IfIndex:    15,
				Name:       "EPON0/1",
				OperStatus: 1,
			},
		},
	}

	got, err := (BDCOMONUAdapter{}).
		BuildPersistenceCandidates(
			ifmib,
			nil,
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
		got[0].ONUNo != 1 ||
		got[0].IfIndex != 84 ||
		got[0].OperStatus != "UP" ||
		got[0].InOctets != in ||
		got[0].OutOctets != out {
		t.Fatalf(
			"unexpected candidate: %+v",
			got[0],
		)
	}
}

func TestBDCOMONUAdapterMatches(t *testing.T) {
	adapter := BDCOMONUAdapter{}

	if !adapter.Matches(
		"BDCOM",
		"",
	) {
		t.Fatal(
			"BDCOM vendor must match adapter",
		)
	}

	if !adapter.Matches(
		"",
		".1.3.6.1.4.1.3320.1",
	) {
		t.Fatal(
			"BDCOM enterprise sysObjectID must match adapter",
		)
	}
}

func TestMergeBDCOMONUInventory(t *testing.T) {
	candidates := []ONUPersistenceCandidate{
		{
			PONNo:      1,
			ONUNo:      1,
			IfIndex:    84,
			OperStatus: "UNKNOWN",
		},
		{
			PONNo:      1,
			ONUNo:      2,
			IfIndex:    85,
			OperStatus: "UP",
		},
	}

	inventory := &BDCOMONUInventoryCollection{
		Vendor: "BDCOM",
		Records: []BDCOMONUInventoryRecord{
			{
				IfIndex:    84,
				Model:      "D401",
				MACAddress: "4C:46:D1:55:9E:69",
				OperStatus: "UP",
			},
			{
				IfIndex:    85,
				Model:      "----",
				MACAddress: "4C:46:D1:F5:2C:56",
				OperStatus: "DOWN",
			},
		},
	}

	got := MergeBDCOMONUInventory(candidates, inventory)

	if got[0].MACAddress != "4C:46:D1:55:9E:69" ||
		got[0].Model != "D401" ||
		got[0].OperStatus != "UP" {
		t.Fatalf("unexpected first candidate: %+v", got[0])
	}

	if got[1].MACAddress != "4C:46:D1:F5:2C:56" ||
		got[1].Model != "" ||
		got[1].OperStatus != "DOWN" {
		t.Fatalf("unexpected second candidate: %+v", got[1])
	}
}

func TestParseBDCOMONUOpticalRows(t *testing.T) {
	records, err := ParseBDCOMONUOpticalRows(
		[]WalkResult{
			{
				OID:   BDCOMONUOpticalRxPowerOID + ".33",
				Value: -193,
			},
		},
		[]WalkResult{
			{
				OID:   BDCOMONUOpticalTxPowerOID + ".33",
				Value: 19,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 1 {
		t.Fatalf("record count=%d want=1", len(records))
	}

	got := records[0]

	if got.IfIndex != 33 ||
		got.RxPowerDBM == nil ||
		got.TxPowerDBM == nil {
		t.Fatalf("unexpected record: %+v", got)
	}

	if *got.RxPowerDBM != -19.3 {
		t.Fatalf("RX=%v want=-19.3", *got.RxPowerDBM)
	}

	if *got.TxPowerDBM != 1.9 {
		t.Fatalf("TX=%v want=1.9", *got.TxPowerDBM)
	}
}

func TestMergeBDCOMONUOptical(t *testing.T) {
	rx := -19.3
	tx := 1.9

	candidates := []ONUPersistenceCandidate{
		{
			PONNo:      4,
			ONUNo:      13,
			IfIndex:    33,
			OperStatus: "UP",
			MACAddress: "4C:D7:C8:BD:1A:0C",
		},
	}

	optical := &ONUOpticalCollection{
		Vendor: "BDCOM",
		Records: []ONUOpticalRecord{
			{
				IfIndex:    33,
				RxPowerDBM: &rx,
				TxPowerDBM: &tx,
			},
		},
	}

	got := MergeBDCOMONUOptical(candidates, optical)

	if got[0].RxPowerDBM == nil ||
		got[0].TxPowerDBM == nil ||
		*got[0].RxPowerDBM != -19.3 ||
		*got[0].TxPowerDBM != 1.9 {
		t.Fatalf("unexpected optical merge: %+v", got[0])
	}

	if got[0].MACAddress != "4C:D7:C8:BD:1A:0C" ||
		got[0].OperStatus != "UP" {
		t.Fatalf("existing fields changed: %+v", got[0])
	}
}

func TestParseBDCOMDeciDBMPointerSentinel(t *testing.T) {
	if got := parseBDCOMDeciDBMPointer("-65535"); got != nil {
		t.Fatalf("expected -65535 sentinel to return nil, got %v", *got)
	}

	if got := parseBDCOMDeciDBMPointer("-2147483648"); got != nil {
		t.Fatalf("expected int32 minimum sentinel to return nil, got %v", *got)
	}
}
