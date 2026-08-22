package snmp

import (
	"errors"
	"testing"

	gosnmp "github.com/gosnmp/gosnmp"
)

func TestValidateSingleVarBindAcceptsValue(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error: gosnmp.NoError,
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  SysDescrOID,
				Type:  gosnmp.OctetString,
				Value: []byte("V1600D"),
			},
		},
	}

	result, err := ValidateSingleVarBind(packet)
	if err != nil {
		t.Fatalf("ValidateSingleVarBind() error = %v", err)
	}

	if result == nil {
		t.Fatal("ValidateSingleVarBind() returned nil result")
	}

	if result.OID != SysDescrOID {
		t.Fatalf("result.OID = %q, want %q", result.OID, SysDescrOID)
	}
}

func TestValidateSingleVarBindRejectsNoSuchObject(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error: gosnmp.NoError,
		Variables: []gosnmp.SnmpPDU{
			{
				Name: SysDescrOID,
				Type: gosnmp.NoSuchObject,
			},
		},
	}

	if _, err := ValidateSingleVarBind(packet); err == nil {
		t.Fatal("expected noSuchObject error")
	}
}

func TestValidateSingleVarBindRejectsEndOfMibView(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error: gosnmp.NoError,
		Variables: []gosnmp.SnmpPDU{
			{
				Name: SysDescrOID,
				Type: gosnmp.EndOfMibView,
			},
		},
	}

	if _, err := ValidateSingleVarBind(packet); err == nil {
		t.Fatal("expected endOfMibView error")
	}
}

func TestValidateSingleVarBindRejectsProtocolError(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error:      gosnmp.NoSuchName,
		ErrorIndex: 1,
	}

	if _, err := ValidateSingleVarBind(packet); err == nil {
		t.Fatal("expected SNMP protocol error")
	}
}

func TestValidateSingleVarBindRejectsMultipleVariables(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error: gosnmp.NoError,
		Variables: []gosnmp.SnmpPDU{
			{Name: SysDescrOID, Type: gosnmp.OctetString, Value: []byte("one")},
			{Name: SysNameOID, Type: gosnmp.OctetString, Value: []byte("two")},
		},
	}

	if _, err := ValidateSingleVarBind(packet); err == nil {
		t.Fatal("expected variable count error")
	}
}

func TestNewV2CClientPreservesCommunityWhitespace(t *testing.T) {
	const community = "  ts cloud  "

	client, err := NewV2CClient(V2CConfig{
		Host:      "192.0.2.10",
		Community: community,
	})
	if err != nil {
		t.Fatalf("NewV2CClient() error = %v", err)
	}

	if client.Community != community {
		t.Fatalf(
			"client.Community changed: got %q, want exact original value",
			client.Community,
		)
	}
}

func TestNewV2CClientRejectsWhitespaceOnlyCommunity(t *testing.T) {
	_, err := NewV2CClient(V2CConfig{
		Host:      "192.0.2.10",
		Community: "   ",
	})
	if err == nil {
		t.Fatal("expected whitespace-only community validation error")
	}
}

func TestValidateSingleVarBindReturnsTypedNoSuchObject(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error: gosnmp.NoError,
		Variables: []gosnmp.SnmpPDU{
			{
				Name: SysDescrOID,
				Type: gosnmp.NoSuchObject,
			},
		},
	}

	_, err := ValidateSingleVarBind(packet)
	if err == nil {
		t.Fatal("expected response error")
	}

	if !IsResponseError(err) {
		t.Fatalf("error type = %T, want ResponseError", err)
	}

	kind, ok := ResponseErrorKindOf(err)
	if !ok {
		t.Fatal("ResponseErrorKindOf() did not recognize response error")
	}

	if kind != ResponseNoSuchObject {
		t.Fatalf("response kind = %q, want %q", kind, ResponseNoSuchObject)
	}
}

func TestValidateSingleVarBindReturnsTypedEndOfMibView(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error: gosnmp.NoError,
		Variables: []gosnmp.SnmpPDU{
			{
				Name: SysDescrOID,
				Type: gosnmp.EndOfMibView,
			},
		},
	}

	_, err := ValidateSingleVarBind(packet)
	if err == nil {
		t.Fatal("expected response error")
	}

	kind, ok := ResponseErrorKindOf(err)
	if !ok {
		t.Fatal("ResponseErrorKindOf() did not recognize response error")
	}

	if kind != ResponseEndOfMibView {
		t.Fatalf("response kind = %q, want %q", kind, ResponseEndOfMibView)
	}
}

func TestValidateSingleVarBindReturnsTypedProtocolError(t *testing.T) {
	packet := &gosnmp.SnmpPacket{
		Error:      gosnmp.NoSuchName,
		ErrorIndex: 1,
	}

	_, err := ValidateSingleVarBind(packet)
	if err == nil {
		t.Fatal("expected response error")
	}

	kind, ok := ResponseErrorKindOf(err)
	if !ok {
		t.Fatal("ResponseErrorKindOf() did not recognize response error")
	}

	if kind != ResponseProtocolError {
		t.Fatalf("response kind = %q, want %q", kind, ResponseProtocolError)
	}
}

func TestTransportErrorClassification(t *testing.T) {
	err := &TransportError{
		Operation: "GET",
		Err:       errors.New("timeout"),
	}

	if !IsTransportError(err) {
		t.Fatal("IsTransportError() = false, want true")
	}

	if IsResponseError(err) {
		t.Fatal("transport error must not be classified as response error")
	}
}

func TestWalkSubtreeRejectsNilClient(t *testing.T) {
	_, err := WalkSubtree(nil, ".1.3.6.1.2.1.2.2.1.2")
	if err == nil {
		t.Fatal("expected nil client error")
	}
}

func TestWalkSubtreeRejectsBlankOID(t *testing.T) {
	client := &gosnmp.GoSNMP{}

	_, err := WalkSubtree(client, "   ")
	if err == nil {
		t.Fatal("expected blank OID error")
	}
}
