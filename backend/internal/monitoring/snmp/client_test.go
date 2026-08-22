package snmp

import (
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
