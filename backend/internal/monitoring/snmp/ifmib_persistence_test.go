package snmp

import (
	"math"
	"testing"
	"time"
)

func TestBuildPortPersistenceCandidate(t *testing.T) {
	in := uint64(1000)
	out := uint64(2000)
	inErrors := uint64(3)
	inDiscards := uint64(4)

	sampledAt := time.Date(
		2026,
		8,
		23,
		5,
		45,
		0,
		0,
		time.UTC,
	)

	collection := &IFMIBCollection{
		SampledAt:      sampledAt,
		SysUpTimeTicks: 10000,
	}

	port := IFMIBPort{
		IfIndex:         8,
		Name:            "eth0/0/8",
		Description:     "Wabda_OLT-UpLink",
		IfType:          6,
		AdminStatus:     1,
		OperStatus:      1,
		HighSpeedMbps:   1000,
		MACAddress:      "F4:70:0C:87:2A:5B",
		LastChangeTicks: 7000,
		HCInOctets:      &in,
		HCOutOctets:     &out,
		InErrors:        &inErrors,
		InDiscards:      &inDiscards,
	}

	got, err := BuildPortPersistenceCandidate(
		collection,
		port,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.PortKey != "ifindex:8" {
		t.Fatalf(
			"unexpected port key %q",
			got.PortKey,
		)
	}

	if got.PortType != "ETHERNET" {
		t.Fatalf(
			"unexpected port type %q",
			got.PortType,
		)
	}

	if got.AdminStatus != "UP" ||
		got.OperStatus != "UP" {
		t.Fatalf(
			"unexpected status admin=%q oper=%q",
			got.AdminStatus,
			got.OperStatus,
		)
	}

	if got.SpeedMbps != 1000 {
		t.Fatalf(
			"unexpected speed %d",
			got.SpeedMbps,
		)
	}

	if got.InOctets != 1000 ||
		got.OutOctets != 2000 {
		t.Fatal("unexpected octet values")
	}

	if got.OutErrors != 0 {
		t.Fatalf(
			"missing optional out errors must map to 0, got %d",
			got.OutErrors,
		)
	}

	if got.OutDiscards != 0 {
		t.Fatalf(
			"missing optional out discards must map to 0, got %d",
			got.OutDiscards,
		)
	}

	if got.LastChangeAt == nil {
		t.Fatal("expected last change timestamp")
	}
}

func TestBuildPortPersistenceCandidateLogicalVLAN(
	t *testing.T,
) {
	in := uint64(10)
	out := uint64(20)

	collection := &IFMIBCollection{
		SampledAt: time.Now(),
	}

	port := IFMIBPort{
		IfIndex:     17825794,
		Name:        "VLAN-IF20",
		IfType:      1,
		AdminStatus: 1,
		OperStatus:  1,
		HCInOctets:  &in,
		HCOutOctets: &out,
	}

	got, err := BuildPortPersistenceCandidate(
		collection,
		port,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.PortKey != "ifindex:17825794" {
		t.Fatalf(
			"unexpected port key %q",
			got.PortKey,
		)
	}

	if got.PortType != "VLAN" {
		t.Fatalf(
			"unexpected port type %q",
			got.PortType,
		)
	}
}

func TestBuildPortPersistenceCandidateRequiresHCIn(
	t *testing.T,
) {
	out := uint64(20)

	_, err := BuildPortPersistenceCandidate(
		&IFMIBCollection{
			SampledAt: time.Now(),
		},
		IFMIBPort{
			IfIndex:     1,
			HCOutOctets: &out,
		},
	)

	if err == nil {
		t.Fatal("expected missing HC in error")
	}
}

func TestBuildPortPersistenceCandidateRequiresHCOut(
	t *testing.T,
) {
	in := uint64(10)

	_, err := BuildPortPersistenceCandidate(
		&IFMIBCollection{
			SampledAt: time.Now(),
		},
		IFMIBPort{
			IfIndex:    1,
			HCInOctets: &in,
		},
	)

	if err == nil {
		t.Fatal("expected missing HC out error")
	}
}

func TestBuildPortPersistenceCandidatesPreservesOrder(
	t *testing.T,
) {
	in1 := uint64(1)
	out1 := uint64(2)
	in2 := uint64(3)
	out2 := uint64(4)

	collection := &IFMIBCollection{
		SampledAt: time.Now(),
		Ports: []IFMIBPort{
			{
				IfIndex:     1,
				HCInOctets:  &in1,
				HCOutOctets: &out1,
			},
			{
				IfIndex:     8,
				HCInOctets:  &in2,
				HCOutOctets: &out2,
			},
		},
	}

	got, err := BuildPortPersistenceCandidates(
		collection,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf(
			"expected 2 candidates got %d",
			len(got),
		)
	}

	if got[0].IfIndex != 1 ||
		got[1].IfIndex != 8 {
		t.Fatalf(
			"unexpected order %+v",
			got,
		)
	}
}

func TestBuildPortSampleRateCandidate(t *testing.T) {
	previousAt := time.Date(
		2026,
		8,
		23,
		5,
		50,
		0,
		0,
		time.UTC,
	)

	current := PortPersistenceCandidate{
		InOctets:  2_000_000,
		OutOctets: 5_000_000,
		SampledAt: previousAt.Add(
			10 * time.Second,
		),
	}

	got := BuildPortSampleRateCandidate(
		previousAt,
		1_000_000,
		3_000_000,
		current,
	)

	if math.Abs(got.InMbps-0.8) > 0.000001 {
		t.Fatalf(
			"expected in 0.8 Mbps got %.6f",
			got.InMbps,
		)
	}

	if math.Abs(got.OutMbps-1.6) > 0.000001 {
		t.Fatalf(
			"expected out 1.6 Mbps got %.6f",
			got.OutMbps,
		)
	}
}

func TestBuildPortSampleRateCandidateCounterReset(
	t *testing.T,
) {
	previousAt := time.Now()

	current := PortPersistenceCandidate{
		InOctets:  100,
		OutOctets: 200,
		SampledAt: previousAt.Add(
			10 * time.Second,
		),
	}

	got := BuildPortSampleRateCandidate(
		previousAt,
		500,
		600,
		current,
	)

	if got.InMbps != 0 {
		t.Fatalf(
			"expected reset in rate 0 got %f",
			got.InMbps,
		)
	}

	if got.OutMbps != 0 {
		t.Fatalf(
			"expected reset out rate 0 got %f",
			got.OutMbps,
		)
	}
}

func TestBuildPortSampleRateCandidateZeroElapsed(
	t *testing.T,
) {
	now := time.Now()

	current := PortPersistenceCandidate{
		InOctets:  200,
		OutOctets: 300,
		SampledAt: now,
	}

	got := BuildPortSampleRateCandidate(
		now,
		100,
		200,
		current,
	)

	if got.InMbps != 0 ||
		got.OutMbps != 0 {
		t.Fatalf(
			"expected zero rates got in=%f out=%f",
			got.InMbps,
			got.OutMbps,
		)
	}
}
