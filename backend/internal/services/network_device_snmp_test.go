package services

import (
	"errors"
	"testing"

	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
)

func TestClassifySNMPProbeOutcomeOnline(t *testing.T) {
	status := classifySNMPProbeOutcome(true, nil)

	if status != "ONLINE" {
		t.Fatalf("status = %q, want ONLINE", status)
	}
}

func TestClassifySNMPProbeOutcomeDegradedForResponseError(t *testing.T) {
	status := classifySNMPProbeOutcome(false, []error{
		&snmpmonitor.ResponseError{
			Kind:   snmpmonitor.ResponseNoSuchObject,
			Detail: "OID returned noSuchObject",
		},
	})

	if status != "DEGRADED" {
		t.Fatalf("status = %q, want DEGRADED", status)
	}
}

func TestClassifySNMPProbeOutcomeOfflineForTransportError(t *testing.T) {
	status := classifySNMPProbeOutcome(false, []error{
		&snmpmonitor.TransportError{
			Operation: "GET",
			Err:       errors.New("timeout"),
		},
	})

	if status != "OFFLINE" {
		t.Fatalf("status = %q, want OFFLINE", status)
	}
}

func TestClassifySNMPProbeOutcomePrefersDegradedWhenAgentResponded(t *testing.T) {
	status := classifySNMPProbeOutcome(false, []error{
		&snmpmonitor.TransportError{
			Operation: "GET",
			Err:       errors.New("timeout"),
		},
		&snmpmonitor.ResponseError{
			Kind:   snmpmonitor.ResponseEndOfMibView,
			Detail: "OID returned endOfMibView",
		},
	})

	if status != "DEGRADED" {
		t.Fatalf("status = %q, want DEGRADED", status)
	}
}

func TestClassifySNMPProbeOutcomeUnknownFailureDefaultsOffline(t *testing.T) {
	status := classifySNMPProbeOutcome(false, []error{
		errors.New("unexpected failure"),
	})

	if status != "OFFLINE" {
		t.Fatalf("status = %q, want OFFLINE", status)
	}
}
