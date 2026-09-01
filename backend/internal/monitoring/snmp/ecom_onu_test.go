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

func TestParseECOMONUOpticalRows(t *testing.T) {
	got, err := ParseECOMONUOpticalRows(
		[]WalkResult{
			{
				OID:   ECOMONUOpticalRxPowerOID + ".21495809.1.1",
				Value: -233,
			},
			{
				OID:   ECOMONUOpticalRxPowerOID + ".21499907.1.1",
				Value: -201,
			},
		},
		[]WalkResult{
			{
				OID:   ECOMONUOpticalTxPowerOID + ".21495809.1.1",
				Value: 145,
			},
			{
				OID:   ECOMONUOpticalTxPowerOID + ".21499907.1.1",
				Value: 132,
			},
		},
	)
	if err != nil {
		t.Fatalf("parse optical rows: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}

	if got[0].IfIndex != 21495809 ||
		got[0].RxPowerDBM == nil || *got[0].RxPowerDBM != -2.33 ||
		got[0].TxPowerDBM == nil || *got[0].TxPowerDBM != 1.45 {
		t.Fatalf("unexpected first record: %+v", got[0])
	}
}

func TestParseECOMONUOpticalOIDRejectsNonCanonicalSuffix(t *testing.T) {
	if _, ok := parseECOMONUOpticalOID(
		ECOMONUOpticalRxPowerOID+".21495809.1.2",
		ECOMONUOpticalRxPowerOID,
	); ok {
		t.Fatal("expected non-canonical suffix rejection")
	}
}

func TestMergeECOMONUOptical(t *testing.T) {
	txPower := 1.45
	rxPower := -2.33

	got := MergeECOMONUOptical(
		[]ONUPersistenceCandidate{
			{
				PONNo:   1,
				ONUNo:   1,
				IfIndex: 21495809,
			},
			{
				PONNo:   2,
				ONUNo:   3,
				IfIndex: 21499907,
			},
		},
		&ONUOpticalCollection{
			Vendor: "ECOM",
			Records: []ONUOpticalRecord{
				{
					IfIndex:    21495809,
					TxPowerDBM: &txPower,
					RxPowerDBM: &rxPower,
				},
			},
		},
	)

	if got[0].TxPowerDBM == nil || *got[0].TxPowerDBM != 1.45 ||
		got[0].RxPowerDBM == nil || *got[0].RxPowerDBM != -2.33 {
		t.Fatalf("expected optical values on matching candidate: %+v", got[0])
	}

	if got[1].TxPowerDBM != nil || got[1].RxPowerDBM != nil {
		t.Fatalf("unexpected optical values on non-matching candidate: %+v", got[1])
	}
}

func TestParseECOMSniMACRecordLiveShape(t *testing.T) {
	record, err := ParseECOMSniMACRecord(
		ECOMSniMacAddressTypeOID+
			".6.4.149.230.88.142.232.3501",
		2,
		ECOMSniMacAddressPortOID+
			".6.4.149.230.88.142.232.3501",
		17825793,
	)
	if err != nil {
		t.Fatalf("parse ECOM SNI MAC record: %v", err)
	}

	if record.MACAddress != "04:95:E6:58:8E:E8" ||
		record.VLAN != 3501 ||
		record.MACType != 2 ||
		record.PortID != 17825793 {
		t.Fatalf("unexpected ECOM SNI MAC record: %+v", record)
	}
}

func TestParseECOMSniMACRecordRejectsMismatchedIndexes(
	t *testing.T,
) {
	_, err := ParseECOMSniMACRecord(
		ECOMSniMacAddressTypeOID+
			".6.4.149.230.88.142.232.3501",
		2,
		ECOMSniMacAddressPortOID+
			".6.4.149.230.88.142.232.3502",
		17825793,
	)

	if err == nil {
		t.Fatal("expected mismatched ECOM SNI MAC index error")
	}
}

func TestParseECOMSniMACIndexRejectsMalformedMAC(t *testing.T) {
	_, _, ok := parseECOMSniMACIndex(
		ECOMSniMacAddressTypeOID+
			".6.4.149.999.88.142.232.3501",
		ECOMSniMacAddressTypeOID,
	)

	if ok {
		t.Fatal("expected malformed ECOM SNI MAC index rejection")
	}
}

func TestParseECOMSniMACRowsAndFindLiveMAC(t *testing.T) {
	suffix := ".6.4.149.230.88.142.232.3501"

	records, err := ParseECOMSniMACRows(
		[]WalkResult{{
			OID:   ECOMSniMacAddressTypeOID + suffix,
			Value: 2,
		}},
		[]WalkResult{{
			OID:   ECOMSniMacAddressPortOID + suffix,
			Value: uint32(17825793),
		}},
	)
	if err != nil {
		t.Fatalf("parse ECOM SNI MAC rows: %v", err)
	}

	record, ok := FindECOMLearnedMAC(
		records,
		"04:95:e6:58:8e:e8",
	)
	if !ok {
		t.Fatal("expected live learned MAC match")
	}

	if record.MACAddress != "04:95:E6:58:8E:E8" ||
		record.VLAN != 3501 ||
		record.MACType != 2 ||
		record.PortID != 17825793 {
		t.Fatalf("unexpected learned MAC: %+v", record)
	}
}

func TestParseECOMSniMACRowsRequiresMatchingColumns(t *testing.T) {
	records, err := ParseECOMSniMACRows(
		[]WalkResult{{
			OID: ECOMSniMacAddressTypeOID +
				".6.4.149.230.88.142.232.3501",
			Value: 2,
		}},
		[]WalkResult{{
			OID: ECOMSniMacAddressPortOID +
				".6.4.149.230.88.142.232.3502",
			Value: uint32(17825793),
		}},
	)
	if err != nil {
		t.Fatalf("parse ECOM SNI MAC rows: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("expected no joined records, got %+v", records)
	}
}

func TestParseECOMPONInterface(t *testing.T) {
	ponNo, ok := ParseECOMPONInterface("epon 0/1/1")
	if !ok || ponNo != 1 {
		t.Fatalf("PON=%d ok=%v want=1,true", ponNo, ok)
	}

	if _, ok := ParseECOMPONInterface("xge 0/0/1"); ok {
		t.Fatal("expected XGE interface rejection")
	}

	if _, ok := ParseECOMPONInterface(
		"epon 0/1/1 onu 1",
	); ok {
		t.Fatal("expected ONU interface rejection")
	}
}

func TestResolveECOMLearnedMACLiveShape(t *testing.T) {
	suffix := ".6.4.149.230.88.142.232.3501"

	walk := func(rootOID string) ([]WalkResult, error) {
		switch rootOID {
		case ECOMSniMacAddressTypeOID:
			return []WalkResult{{
				OID:   rootOID + suffix,
				Value: 2,
			}}, nil

		case ECOMSniMacAddressPortOID:
			return []WalkResult{{
				OID:   rootOID + suffix,
				Value: uint32(17825793),
			}}, nil

		default:
			t.Fatalf("unexpected walk OID %s", rootOID)
			return nil, nil
		}
	}

	get := func(oid string) (*ProbeResult, error) {
		want := IFNameOID + ".17825793"
		if oid != want {
			t.Fatalf("GET OID=%s want=%s", oid, want)
		}

		return &ProbeResult{
			OID:   oid,
			Value: []byte("epon 0/1/1"),
		}, nil
	}

	got, err := resolveECOMLearnedMAC(
		"04:95:E6:58:8E:E8",
		walk,
		get,
	)
	if err != nil {
		t.Fatalf("resolve learned MAC: %v", err)
	}
	if got == nil {
		t.Fatal("expected learned MAC resolution")
	}

	if got.MACAddress != "04:95:E6:58:8E:E8" ||
		got.VLAN != 3501 ||
		got.MACType != 2 ||
		got.PortID != 17825793 ||
		got.Interface != "epon 0/1/1" ||
		got.PONNo != 1 {
		t.Fatalf("unexpected resolution: %+v", got)
	}
}

func TestResolveECOMLearnedMACNotFound(t *testing.T) {
	walk := func(rootOID string) ([]WalkResult, error) {
		return []WalkResult{}, nil
	}

	get := func(oid string) (*ProbeResult, error) {
		t.Fatalf("GET must not run when MAC is not learned")
		return nil, nil
	}

	got, err := resolveECOMLearnedMAC(
		"04:95:E6:58:8E:E8",
		walk,
		get,
	)
	if err != nil {
		t.Fatalf("resolve learned MAC: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil resolution, got %+v", got)
	}
}

func TestResolveECOMLearnedMACRejectsNonPONPort(t *testing.T) {
	suffix := ".6.4.149.230.88.142.232.798"

	walk := func(rootOID string) ([]WalkResult, error) {
		switch rootOID {
		case ECOMSniMacAddressTypeOID:
			return []WalkResult{{
				OID:   rootOID + suffix,
				Value: 2,
			}}, nil
		case ECOMSniMacAddressPortOID:
			return []WalkResult{{
				OID:   rootOID + suffix,
				Value: uint32(524289),
			}}, nil
		default:
			return nil, nil
		}
	}

	get := func(oid string) (*ProbeResult, error) {
		return &ProbeResult{
			OID:   oid,
			Value: []byte("xge 0/0/1"),
		}, nil
	}

	got, err := resolveECOMLearnedMAC(
		"04:95:E6:58:8E:E8",
		walk,
		get,
	)
	if err == nil {
		t.Fatalf("expected non-PON interface error, got %+v", got)
	}
}
