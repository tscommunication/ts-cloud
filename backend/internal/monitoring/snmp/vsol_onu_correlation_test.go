package snmp

import (
	"math"
	"testing"
	"time"
)

func TestParseVSOLIFMIBONUName(
	t *testing.T,
) {
	tests := []struct {
		name string
		pon  int
		onu  int
		ok   bool
	}{
		{
			name: "EPON01ONU1",
			pon:  1,
			onu:  1,
			ok:   true,
		},
		{
			name: "EPON02ONU48",
			pon:  2,
			onu:  48,
			ok:   true,
		},
		{
			name: "epon03onu51",
			pon:  3,
			onu:  51,
			ok:   true,
		},
		{
			name: "EPON0/1",
			ok:   false,
		},
		{
			name: "GE0/1",
			ok:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pon, onu, ok :=
				ParseVSOLIFMIBONUName(test.name)

			if ok != test.ok {
				t.Fatalf(
					"ok=%v want=%v",
					ok,
					test.ok,
				)
			}

			if !ok {
				return
			}

			if pon != test.pon ||
				onu != test.onu {
				t.Fatalf(
					"PON=%d ONU=%d want PON=%d ONU=%d",
					pon,
					onu,
					test.pon,
					test.onu,
				)
			}
		})
	}
}

func TestBuildVSOLONUPersistenceCandidatesMergesIFMIBAndOptical(
	t *testing.T,
) {
	sampledAt := time.Date(
		2026,
		time.August,
		23,
		7,
		30,
		0,
		0,
		time.UTC,
	)

	in := uint64(1_000_000)
	out := uint64(2_000_000)

	temp := 35.00
	voltage := 3.22
	bias := 15.00
	tx := 2.22
	rx := -12.64

	ifmib := &IFMIBCollection{
		SampledAt: sampledAt,
		Ports: []IFMIBPort{
			{
				IfIndex:     14,
				Name:        "EPON01ONU2",
				OperStatus:  1,
				MACAddress:  "E0:67:B3:11:22:33",
				HCInOctets:  &in,
				HCOutOctets: &out,
			},
			{
				IfIndex:    9,
				Name:       "EPON0/1",
				OperStatus: 1,
			},
		},
	}

	optical := &ONUOpticalCollection{
		Vendor:    "VSOL",
		SampledAt: sampledAt,
		Records: []ONUOpticalRecord{
			{
				PONNo:        1,
				ONUNo:        2,
				TemperatureC: &temp,
				VoltageV:     &voltage,
				TxBiasMA:     &bias,
				TxPowerDBM:   &tx,
				RxPowerDBM:   &rx,
			},
		},
	}

	got, err :=
		BuildVSOLONUPersistenceCandidates(
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

	candidate := got[0]

	if candidate.PONNo != 1 ||
		candidate.ONUNo != 2 ||
		candidate.IfIndex != 14 {
		t.Fatalf(
			"unexpected key/index PON=%d ONU=%d ifIndex=%d",
			candidate.PONNo,
			candidate.ONUNo,
			candidate.IfIndex,
		)
	}

	if candidate.MACAddress != "E0:67:B3:11:22:33" {
		t.Fatalf(
			"MAC=%q want=%q",
			candidate.MACAddress,
			"E0:67:B3:11:22:33",
		)
	}

	if candidate.OperStatus != "UP" {
		t.Fatalf(
			"oper status=%q want=UP",
			candidate.OperStatus,
		)
	}

	if candidate.InOctets != in ||
		candidate.OutOctets != out {
		t.Fatalf(
			"unexpected counters in=%d out=%d",
			candidate.InOctets,
			candidate.OutOctets,
		)
	}

	assertCandidateFloat(
		t,
		"temperature",
		candidate.TemperatureC,
		temp,
	)
	assertCandidateFloat(
		t,
		"voltage",
		candidate.VoltageV,
		voltage,
	)
	assertCandidateFloat(
		t,
		"bias",
		candidate.TxBiasMA,
		bias,
	)
	assertCandidateFloat(
		t,
		"tx",
		candidate.TxPowerDBM,
		tx,
	)
	assertCandidateFloat(
		t,
		"rx",
		candidate.RxPowerDBM,
		rx,
	)
}

func TestBuildVSOLONUPersistenceCandidatesKeepsOfflineBlankOptics(
	t *testing.T,
) {
	sampledAt := time.Now()

	in := uint64(100)
	out := uint64(200)

	ifmib := &IFMIBCollection{
		SampledAt: sampledAt,
		Ports: []IFMIBPort{
			{
				IfIndex:     172,
				Name:        "EPON03ONU10",
				OperStatus:  2,
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
				PONNo: 3,
				ONUNo: 10,
			},
		},
	}

	got, err :=
		BuildVSOLONUPersistenceCandidates(
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

	candidate := got[0]

	if candidate.OperStatus != "DOWN" {
		t.Fatalf(
			"status=%q want=DOWN",
			candidate.OperStatus,
		)
	}

	if candidate.RxPowerDBM != nil ||
		candidate.TxPowerDBM != nil ||
		candidate.TemperatureC != nil {
		t.Fatal(
			"offline blank optical values must remain nil",
		)
	}
}

func TestBuildVSOLONUPersistenceCandidatesAllowsMissingIFMIBMatch(
	t *testing.T,
) {
	sampledAt := time.Now()
	rx := -18.50

	optical := &ONUOpticalCollection{
		Vendor:    "VSOL",
		SampledAt: sampledAt,
		Records: []ONUOpticalRecord{
			{
				PONNo:      1,
				ONUNo:      5,
				RxPowerDBM: &rx,
			},
		},
	}

	got, err :=
		BuildVSOLONUPersistenceCandidates(
			nil,
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

	if got[0].OperStatus != "UNKNOWN" ||
		got[0].IfIndex != 0 {
		t.Fatalf(
			"unexpected unmatched candidate status=%q ifIndex=%d",
			got[0].OperStatus,
			got[0].IfIndex,
		)
	}
}

func TestBuildVSOLONUPersistenceCandidatesRejectsDuplicateIFMIBMapping(
	t *testing.T,
) {
	sampledAt := time.Now()

	ifmib := &IFMIBCollection{
		SampledAt: sampledAt,
		Ports: []IFMIBPort{
			{
				IfIndex: 10,
				Name:    "EPON01ONU2",
			},
			{
				IfIndex: 11,
				Name:    "EPON01ONU2",
			},
		},
	}

	optical := &ONUOpticalCollection{
		SampledAt: sampledAt,
		Records: []ONUOpticalRecord{
			{
				PONNo: 1,
				ONUNo: 2,
			},
		},
	}

	if _, err :=
		BuildVSOLONUPersistenceCandidates(
			ifmib,
			optical,
		); err == nil {
		t.Fatal(
			"expected duplicate IF-MIB mapping error",
		)
	}
}

func assertCandidateFloat(
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
			"%s=%f want=%f",
			name,
			*got,
			want,
		)
	}
}

func TestBuildVSOLONUPersistenceCandidatesFallsBackToIFMIB(
	t *testing.T,
) {
	sampledAt := time.Date(
		2026,
		time.August,
		23,
		12,
		30,
		0,
		0,
		time.UTC,
	)

	in1 := uint64(1000)
	out1 := uint64(2000)
	in2 := uint64(3000)
	out2 := uint64(4000)

	ifmib := &IFMIBCollection{
		SampledAt: sampledAt,
		Ports: []IFMIBPort{
			{
				IfIndex:     22,
				Name:        "EPON02ONU1",
				OperStatus:  2,
				HCInOctets:  &in2,
				HCOutOctets: &out2,
			},
			{
				IfIndex:     11,
				Name:        "EPON01ONU2",
				OperStatus:  1,
				HCInOctets:  &in1,
				HCOutOctets: &out1,
			},
		},
	}

	got, err :=
		BuildVSOLONUPersistenceCandidates(
			ifmib,
			nil,
		)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf(
			"candidate count=%d want=2",
			len(got),
		)
	}

	first := got[0]

	if first.PONNo != 1 ||
		first.ONUNo != 2 ||
		first.IfIndex != 11 ||
		first.OperStatus != "UP" {
		t.Fatalf(
			"unexpected first candidate: %+v",
			first,
		)
	}

	if first.InOctets != in1 ||
		first.OutOctets != out1 {
		t.Fatalf(
			"unexpected first counters in=%d out=%d",
			first.InOctets,
			first.OutOctets,
		)
	}

	if first.RxPowerDBM != nil ||
		first.TxPowerDBM != nil ||
		first.TemperatureC != nil ||
		first.VoltageV != nil ||
		first.TxBiasMA != nil {
		t.Fatal(
			"IF-MIB fallback must not invent optical values",
		)
	}

	if !first.SampledAt.Equal(sampledAt) {
		t.Fatalf(
			"sample time=%v want=%v",
			first.SampledAt,
			sampledAt,
		)
	}

	second := got[1]

	if second.PONNo != 2 ||
		second.ONUNo != 1 ||
		second.IfIndex != 22 ||
		second.OperStatus != "DOWN" {
		t.Fatalf(
			"unexpected second candidate: %+v",
			second,
		)
	}
}

func TestParseVSOLIFMIBONUNameSupportsLegacyAndSlashFormats(
	t *testing.T,
) {
	tests := []struct {
		name    string
		input   string
		wantPON int
		wantONU int
		wantOK  bool
	}{
		{
			name:    "legacy exact",
			input:   "EPON01ONU23",
			wantPON: 1,
			wantONU: 23,
			wantOK:  true,
		},
		{
			name:    "legacy with description",
			input:   "EPON02ONU7 Customer_Name",
			wantPON: 2,
			wantONU: 7,
			wantOK:  true,
		},
		{
			name:    "slash exact",
			input:   "EPON0/1:23",
			wantPON: 1,
			wantONU: 23,
			wantOK:  true,
		},
		{
			name:    "slash fourth PON",
			input:   "EPON0/4:54",
			wantPON: 4,
			wantONU: 54,
			wantOK:  true,
		},
		{
			name:   "physical PON is not ONU",
			input:  "EPON0/1",
			wantOK: false,
		},
		{
			name:   "VLAN is not ONU",
			input:  "VLAN631",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPON, gotONU, gotOK :=
				ParseVSOLIFMIBONUName(tt.input)

			if gotOK != tt.wantOK {
				t.Fatalf(
					"ok=%v want=%v",
					gotOK,
					tt.wantOK,
				)
			}

			if !tt.wantOK {
				return
			}

			if gotPON != tt.wantPON ||
				gotONU != tt.wantONU {
				t.Fatalf(
					"PON/ONU=%d/%d want=%d/%d",
					gotPON,
					gotONU,
					tt.wantPON,
					tt.wantONU,
				)
			}
		})
	}
}

func TestBuildVSOLONUPersistenceCandidatesSupportsSlashONUFormat(
	t *testing.T,
) {
	sampledAt := time.Date(
		2026,
		time.August,
		23,
		13,
		30,
		0,
		0,
		time.UTC,
	)

	in := uint64(123456)
	out := uint64(654321)

	ifmib := &IFMIBCollection{
		SampledAt: sampledAt,
		Ports: []IFMIBPort{
			{
				IfIndex:     16,
				Name:        "EPON0/1:1",
				Description: "EPON0/1:1",
				OperStatus:  1,
				HCInOctets:  &in,
				HCOutOctets: &out,
			},
			{
				IfIndex:    9,
				Name:       "EPON0/1",
				OperStatus: 1,
			},
		},
	}

	got, err :=
		BuildVSOLONUPersistenceCandidates(
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

	candidate := got[0]

	if candidate.PONNo != 1 ||
		candidate.ONUNo != 1 ||
		candidate.IfIndex != 16 ||
		candidate.OperStatus != "UP" {
		t.Fatalf(
			"unexpected candidate: %+v",
			candidate,
		)
	}

	if candidate.InOctets != in ||
		candidate.OutOctets != out {
		t.Fatalf(
			"unexpected counters in=%d out=%d",
			candidate.InOctets,
			candidate.OutOctets,
		)
	}

	if candidate.RxPowerDBM != nil ||
		candidate.TxPowerDBM != nil {
		t.Fatal(
			"slash-format IF-MIB fallback must not invent optical values",
		)
	}
}
