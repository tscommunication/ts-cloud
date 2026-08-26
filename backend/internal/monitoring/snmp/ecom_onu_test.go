package snmp

import (
	"testing"
	"time"
)

func TestECOMONUAdapterMatches(t *testing.T) {
	adapter := ECOMONUAdapter{}

	if !adapter.Matches("ECOM", "") {
		t.Fatal("expected ECOM vendor match")
	}

	if !adapter.Matches(
		"",
		".1.3.6.1.4.1.17409",
	) {
		t.Fatal("expected enterprise OID match")
	}

	if adapter.Matches(
		"VSOL",
		".1.3.6.1.4.1.37950",
	) {
		t.Fatal("unexpected VSOL match")
	}
}

func TestParseECOMONUDescription(t *testing.T) {
	pon, onu, ok := parseECOMONUDescription(
		"epon 0/1/3 onu 17 customer_name",
	)

	if !ok || pon != 3 || onu != 17 {
		t.Fatalf(
			"unexpected parse: pon=%d onu=%d ok=%v",
			pon,
			onu,
			ok,
		)
	}
}

func TestParseECOMONUStatus(t *testing.T) {
	if got := parseECOMONUStatus("1"); got != "UP" {
		t.Fatalf("status 1 = %q", got)
	}

	if got := parseECOMONUStatus("2"); got != "DOWN" {
		t.Fatalf("status 2 = %q", got)
	}

	if got := parseECOMONUStatus("9"); got != "UNKNOWN" {
		t.Fatalf("status 9 = %q", got)
	}
}

func TestParseECOMMACValue(t *testing.T) {
	got := parseECOMMACValue(
		[]byte{0xD0, 0x5F, 0xAF, 0x88, 0xDE, 0x68},
	)

	if got != "D0:5F:AF:88:DE:68" {
		t.Fatalf("unexpected MAC %q", got)
	}

	if got := parseECOMMACValue([]byte{1, 2}); got != "" {
		t.Fatalf("expected blank invalid MAC, got %q", got)
	}
}

func TestBuildECOMONUInventoryCandidates(t *testing.T) {
	sampledAt := time.Date(
		2026, 8, 26,
		13, 0, 0, 0,
		time.UTC,
	)

	inventory := &ECOMONUInventoryCollection{
		Vendor:    "ECOM",
		SampledAt: sampledAt,
		Records: []ECOMONUInventoryRecord{
			{
				Index:       21499907,
				PONNo:       2,
				ONUNo:       3,
				Description: "epon 0/1/2 onu 3",
				MACAddress:  "AA:BB:CC:DD:EE:FF",
				OperStatus:  "DOWN",
			},
			{
				Index:       21495809,
				PONNo:       1,
				ONUNo:       1,
				Description: "epon 0/1/1 onu 1",
				MACAddress:  "D0:5F:AF:88:DE:68",
				OperStatus:  "UP",
			},
		},
	}

	got := BuildECOMONUInventoryCandidates(inventory)

	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}

	if got[0].PONNo != 1 ||
		got[0].ONUNo != 1 ||
		got[0].OperStatus != "UP" ||
		got[0].MACAddress != "D0:5F:AF:88:DE:68" {
		t.Fatalf("unexpected first candidate: %+v", got[0])
	}

	if got[1].PONNo != 2 ||
		got[1].ONUNo != 3 ||
		got[1].OperStatus != "DOWN" {
		t.Fatalf("unexpected second candidate: %+v", got[1])
	}
}
