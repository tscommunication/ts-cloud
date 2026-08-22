package snmp

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCollectIFMIBBuildsSortedPorts(t *testing.T) {
	rows := map[string][]WalkResult{
		IFNameOID: {
			{
				OID:   IFNameOID + ".8",
				Value: []byte("eth0/0/8"),
			},
			{
				OID:   IFNameOID + ".1",
				Value: []byte("eth0/0/1"),
			},
		},
		IFDescrOID: {
			{
				OID:   IFDescrOID + ".1",
				Value: []byte("uplink"),
			},
			{
				OID:   IFDescrOID + ".8",
				Value: []byte("OLT"),
			},
		},
		IFAdminStatusOID: {
			{
				OID:   IFAdminStatusOID + ".1",
				Value: int(1),
			},
			{
				OID:   IFAdminStatusOID + ".8",
				Value: int(1),
			},
		},
		IFOperStatusOID: {
			{
				OID:   IFOperStatusOID + ".1",
				Value: int(1),
			},
			{
				OID:   IFOperStatusOID + ".8",
				Value: int(1),
			},
		},
		IFHCInOctetsOID: {
			{
				OID:   IFHCInOctetsOID + ".1",
				Value: uint64(100),
			},
			{
				OID:   IFHCInOctetsOID + ".8",
				Value: uint64(200),
			},
		},
		IFHCOutOctetsOID: {
			{
				OID:   IFHCOutOctetsOID + ".1",
				Value: uint64(300),
			},
			{
				OID:   IFHCOutOctetsOID + ".8",
				Value: uint64(400),
			},
		},
	}

	walk := func(rootOID string) ([]WalkResult, error) {
		return rows[rootOID], nil
	}

	get := func(oid string) (*ProbeResult, error) {
		if oid != SysUpTimeOID {
			return nil, errors.New("unexpected OID")
		}

		return &ProbeResult{
			OID:   SysUpTimeOID,
			Value: uint32(123456),
		}, nil
	}

	sampledAt := time.Date(
		2026,
		8,
		23,
		5,
		40,
		0,
		0,
		time.UTC,
	)

	got, err := collectIFMIB(
		sampledAt,
		walk,
		get,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.SysUpTimeTicks != 123456 {
		t.Fatalf(
			"unexpected uptime %d",
			got.SysUpTimeTicks,
		)
	}

	if len(got.Ports) != 2 {
		t.Fatalf(
			"expected 2 ports, got %d",
			len(got.Ports),
		)
	}

	if got.Ports[0].IfIndex != 1 ||
		got.Ports[1].IfIndex != 8 {
		t.Fatalf(
			"ports not sorted: %+v",
			got.Ports,
		)
	}
}

func TestCollectIFMIBAllowsOptionalColumnFailure(
	t *testing.T,
) {
	requiredRows := func(rootOID string) []WalkResult {
		return []WalkResult{
			{
				OID:   rootOID + ".1",
				Value: uint64(1),
			},
		}
	}

	walk := func(rootOID string) ([]WalkResult, error) {
		switch rootOID {
		case IFNameOID:
			return []WalkResult{
				{
					OID:   IFNameOID + ".1",
					Value: []byte("eth0/0/1"),
				},
			}, nil

		case IFDescrOID:
			return []WalkResult{
				{
					OID:   IFDescrOID + ".1",
					Value: []byte("uplink"),
				},
			}, nil

		case IFAdminStatusOID,
			IFOperStatusOID,
			IFHCInOctetsOID,
			IFHCOutOctetsOID:
			return requiredRows(rootOID), nil

		case IFOutErrorsOID,
			IFOutDiscardsOID:
			return nil, &TransportError{
				Operation: "BULKWALK",
				Err:       errors.New("timeout"),
			}

		default:
			return nil, nil
		}
	}

	get := func(string) (*ProbeResult, error) {
		return &ProbeResult{
			OID:   SysUpTimeOID,
			Value: uint32(1000),
		}, nil
	}

	got, err := collectIFMIB(
		time.Now(),
		walk,
		get,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Warnings) != 2 {
		t.Fatalf(
			"expected 2 warnings, got %d",
			len(got.Warnings),
		)
	}

	if !strings.Contains(
		got.Warnings[0]+got.Warnings[1],
		"optional IF-MIB walk",
	) {
		t.Fatal("expected optional walk warning")
	}
}

func TestCollectIFMIBFailsRequiredColumn(t *testing.T) {
	walk := func(rootOID string) ([]WalkResult, error) {
		if rootOID == IFNameOID {
			return nil, errors.New("failed")
		}

		return nil, nil
	}

	get := func(string) (*ProbeResult, error) {
		return nil, nil
	}

	_, err := collectIFMIB(
		time.Now(),
		walk,
		get,
	)
	if err == nil {
		t.Fatal("expected required column error")
	}

	if !strings.Contains(
		err.Error(),
		"required IF-MIB walk",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectIFMIBAllowsSysUpTimeFailure(
	t *testing.T,
) {
	walk := func(rootOID string) ([]WalkResult, error) {
		switch rootOID {
		case IFNameOID:
			return []WalkResult{
				{
					OID:   rootOID + ".1",
					Value: []byte("eth0/0/1"),
				},
			}, nil

		case IFDescrOID:
			return []WalkResult{
				{
					OID:   rootOID + ".1",
					Value: []byte("uplink"),
				},
			}, nil

		case IFAdminStatusOID,
			IFOperStatusOID,
			IFHCInOctetsOID,
			IFHCOutOctetsOID:
			return []WalkResult{
				{
					OID:   rootOID + ".1",
					Value: uint64(1),
				},
			}, nil
		}

		return nil, nil
	}

	get := func(string) (*ProbeResult, error) {
		return nil, errors.New("timeout")
	}

	got, err := collectIFMIB(
		time.Now(),
		walk,
		get,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.SysUpTimeTicks != 0 {
		t.Fatalf(
			"expected uptime 0, got %d",
			got.SysUpTimeTicks,
		)
	}

	if len(got.Warnings) != 1 {
		t.Fatalf(
			"expected 1 warning, got %d",
			len(got.Warnings),
		)
	}
}
