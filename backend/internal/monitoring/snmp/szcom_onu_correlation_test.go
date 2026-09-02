package snmp

import "testing"

func TestResolveSZCOMLearnedMACPONProductionShape(
	t *testing.T,
) {
	walk := func(
		rootOID string,
	) ([]WalkResult, error) {
		if rootOID != SZCOMDot1dTpFdbPortOID {
			t.Fatalf("unexpected walk OID %q", rootOID)
		}

		return []WalkResult{
			{
				OID: SZCOMDot1dTpFdbPortOID +
					".64.238.21.115.238.201",
				Value: 5002,
			},
		}, nil
	}

	get := func(
		oid string,
	) (*ProbeResult, error) {
		switch oid {
		case SZCOMDot1dBasePortIfIndexOID + ".5002":
			return &ProbeResult{
				OID:   oid,
				Value: 5002,
			}, nil

		case IFNameOID + ".5002":
			return &ProbeResult{
				OID:   oid,
				Value: []byte("pon1"),
			}, nil

		default:
			t.Fatalf("unexpected GET OID %q", oid)
			return nil, nil
		}
	}

	got, err := resolveSZCOMLearnedMACPON(
		"40:EE:15:73:EE:C9",
		walk,
		get,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("expected SZCOM PON resolution")
	}

	if got.MACAddress != "40:EE:15:73:EE:C9" ||
		got.PortID != 5002 ||
		got.IfIndex != 5002 ||
		got.Interface != "pon1" ||
		got.PONNo != 1 {
		t.Fatalf("unexpected resolution: %+v", got)
	}
}

func TestResolveSZCOMLearnedMACPONRejectsUplink(
	t *testing.T,
) {
	walk := func(
		rootOID string,
	) ([]WalkResult, error) {
		return []WalkResult{
			{
				OID: SZCOMDot1dTpFdbPortOID +
					".64.238.21.115.238.201",
				Value: 5012,
			},
		}, nil
	}

	get := func(
		oid string,
	) (*ProbeResult, error) {
		switch oid {
		case SZCOMDot1dBasePortIfIndexOID + ".5012":
			return &ProbeResult{
				OID:   oid,
				Value: 5012,
			}, nil

		case IFNameOID + ".5012":
			return &ProbeResult{
				OID:   oid,
				Value: []byte("xge1"),
			}, nil
		}

		return nil, nil
	}

	got, err := resolveSZCOMLearnedMACPON(
		"40:EE:15:73:EE:C9",
		walk,
		get,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got != nil {
		t.Fatalf("expected soft miss, got %+v", got)
	}
}

func TestSZCOMONUAdapterMatchesCanonicalVendor(
	t *testing.T,
) {
	if !(SZCOMONUAdapter{}).Matches(
		"SZCOM-SOLITINE",
		"",
	) {
		t.Fatal("expected SZCOM-SOLITINE match")
	}
}
