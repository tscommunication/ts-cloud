package snmp

import "testing"

func TestSZCOMONUAdapterMatches(t *testing.T) {
	adapter := SZCOMONUAdapter{}

	if !adapter.Matches("SOLITINE / TBS", "") {
		t.Fatal("expected SOLITINE/TBS vendor match")
	}

	if !adapter.Matches("", ".1.3.6.1.4.1.12170.2.3") {
		t.Fatal("expected SZCOM enterprise OID match")
	}

	if adapter.Matches("HSGQ", ".1.3.6.1.4.1.50224") {
		t.Fatal("unexpected HSGQ match")
	}
}

func TestParseSZCOMONUDeviceIndex(t *testing.T) {
	ponNo, onuNo, ok := parseSZCOMONUDeviceIndex(16777473)

	if !ok || ponNo != 1 || onuNo != 1 {
		t.Fatalf(
			"unexpected device index parse: pon=%d onu=%d ok=%v",
			ponNo,
			onuNo,
			ok,
		)
	}

	if _, _, ok := parseSZCOMONUDeviceIndex(16777472); ok {
		t.Fatal("expected OLT-level index rejection")
	}
}

func TestBuildSZCOMONUInventoryCandidates(t *testing.T) {
	inventory := &SZCOMONUInventoryCollection{
		Vendor: "SZCOM",
		Records: []SZCOMONUInventoryRecord{
			{
				Index:       16777474,
				PONNo:       1,
				ONUNo:       2,
				Description: "onu:1/1:2",
				MACAddress:  "80:F7:A6:B0:AA:A9",
				OperStatus:  "DOWN",
			},
			{
				Index:       16777473,
				PONNo:       1,
				ONUNo:       1,
				Description: "onu:1/1:1",
				MACAddress:  "80:F7:A6:B0:AA:A8",
				OperStatus:  "UP",
			},
		},
	}

	got := BuildSZCOMONUInventoryCandidates(inventory)

	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}

	if got[0].PONNo != 1 ||
		got[0].ONUNo != 1 ||
		got[0].IfIndex != 16777473 ||
		got[0].OperStatus != "UP" ||
		got[0].MACAddress != "80:F7:A6:B0:AA:A8" {
		t.Fatalf("unexpected first candidate: %+v", got[0])
	}
}

func TestResolveONUVendorAdapterSZCOM(t *testing.T) {
	adapter, ok := ResolveONUVendorAdapter(
		"SOLITINE / TBS",
		".1.3.6.1.4.1.12170.2.3",
	)

	if !ok || adapter == nil || adapter.Name() != "SZCOM" {
		t.Fatalf("unexpected resolved adapter: %#v ok=%v", adapter, ok)
	}
}
