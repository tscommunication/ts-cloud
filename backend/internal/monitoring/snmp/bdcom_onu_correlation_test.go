package snmp

import "testing"

func TestParseBDCOMQBridgeFDBPortOIDProductionShape(
	t *testing.T,
) {
	vlan, macAddress, ok :=
		parseBDCOMQBridgeFDBPortOID(
			BDCOMDot1qTpFdbPortOID+
				".624.80.210.245.184.220.78",
			BDCOMDot1qTpFdbPortOID,
		)

	if !ok {
		t.Fatal("expected BDCOM Q-BRIDGE FDB OID to parse")
	}

	if vlan != 624 {
		t.Fatalf("VLAN=%d want=624", vlan)
	}

	if macAddress != "50:D2:F5:B8:DC:4E" {
		t.Fatalf(
			"MAC=%q want=50:D2:F5:B8:DC:4E",
			macAddress,
		)
	}
}

func TestBDCOMFindLearnedMACPort(t *testing.T) {
	rows := []WalkResult{
		{
			OID: BDCOMDot1qTpFdbPortOID +
				".624.80.210.245.184.220.78",
			Value: 284,
		},
	}

	vlan, portID, ok := BDCOMFindLearnedMACPort(
		rows,
		"50:D2:F5:B8:DC:4E",
	)

	if !ok {
		t.Fatal("expected learned MAC match")
	}

	if vlan != 624 || portID != 284 {
		t.Fatalf(
			"VLAN=%d port=%d want VLAN=624 port=284",
			vlan,
			portID,
		)
	}
}

func TestResolveBDCOMLearnedMAC(t *testing.T) {
	walk := func(
		rootOID string,
	) ([]WalkResult, error) {
		if rootOID != BDCOMDot1qTpFdbPortOID {
			t.Fatalf("unexpected root OID %q", rootOID)
		}

		return []WalkResult{
			{
				OID: BDCOMDot1qTpFdbPortOID +
					".624.80.210.245.184.220.78",
				Value: 284,
			},
		}, nil
	}

	get := func(
		oid string,
	) (*ProbeResult, error) {
		if oid != IFNameOID+".284" {
			t.Fatalf("unexpected GET OID %q", oid)
		}

		return &ProbeResult{
			OID:   oid,
			Value: []byte("EPON0/4:8"),
		}, nil
	}

	got, err := resolveBDCOMLearnedMAC(
		"50:D2:F5:B8:DC:4E",
		walk,
		get,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("expected learned MAC resolution")
	}

	if got.MACAddress != "50:D2:F5:B8:DC:4E" ||
		got.VLAN != 624 ||
		got.PortID != 284 ||
		got.Interface != "EPON0/4:8" ||
		got.PONNo != 4 ||
		got.ONUNo != 8 {
		t.Fatalf("unexpected resolution: %+v", got)
	}
}

func TestResolveBDCOMLearnedMACNotFound(t *testing.T) {
	walk := func(
		rootOID string,
	) ([]WalkResult, error) {
		return []WalkResult{}, nil
	}

	get := func(
		oid string,
	) (*ProbeResult, error) {
		t.Fatal("GET must not run without FDB match")
		return nil, nil
	}

	got, err := resolveBDCOMLearnedMAC(
		"50:D2:F5:B8:DC:4E",
		walk,
		get,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got != nil {
		t.Fatalf("expected nil resolution, got %+v", got)
	}
}

func TestResolveBDCOMLearnedMACRejectsNonONUInterface(
	t *testing.T,
) {
	walk := func(
		rootOID string,
	) ([]WalkResult, error) {
		return []WalkResult{
			{
				OID: BDCOMDot1qTpFdbPortOID +
					".624.80.210.245.184.220.78",
				Value: 284,
			},
		}, nil
	}

	get := func(
		oid string,
	) (*ProbeResult, error) {
		return &ProbeResult{
			OID:   oid,
			Value: []byte("GigaEthernet0/1"),
		}, nil
	}

	got, err := resolveBDCOMLearnedMAC(
		"50:D2:F5:B8:DC:4E",
		walk,
		get,
	)

	if err == nil {
		t.Fatal("expected unsupported interface error")
	}

	if got != nil {
		t.Fatalf("expected nil resolution, got %+v", got)
	}
}
