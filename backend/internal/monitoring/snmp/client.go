package snmp

import (
	"errors"
	"fmt"
	"strings"
	"time"

	gosnmp "github.com/gosnmp/gosnmp"
)

const (
	SysDescrOID    = ".1.3.6.1.2.1.1.1.0"
	SysObjectIDOID = ".1.3.6.1.2.1.1.2.0"
	SysNameOID     = ".1.3.6.1.2.1.1.5.0"
)

type V2CConfig struct {
	Host      string
	Port      uint16
	Community string
	Timeout   time.Duration
	Retries   int
}

type ProbeResult struct {
	OID   string
	Type  gosnmp.Asn1BER
	Value interface{}
}

func NewV2CClient(cfg V2CConfig) (*gosnmp.GoSNMP, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return nil, errors.New("SNMP host is required")
	}

	community := strings.TrimSpace(cfg.Community)
	if community == "" {
		return nil, errors.New("SNMP community is required")
	}

	port := cfg.Port
	if port == 0 {
		port = 161
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	retries := cfg.Retries
	if retries < 0 {
		retries = 0
	}

	return &gosnmp.GoSNMP{
		Target:    host,
		Port:      port,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   timeout,
		Retries:   retries,
	}, nil
}

func GetOne(client *gosnmp.GoSNMP, oid string) (*ProbeResult, error) {
	if client == nil {
		return nil, errors.New("SNMP client is nil")
	}

	oid = strings.TrimSpace(oid)
	if oid == "" {
		return nil, errors.New("SNMP OID is required")
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("SNMP connect: %w", err)
	}
	defer client.Conn.Close()

	packet, err := client.Get([]string{oid})
	if err != nil {
		return nil, fmt.Errorf("SNMP GET: %w", err)
	}

	return ValidateSingleVarBind(packet)
}

func ValidateSingleVarBind(packet *gosnmp.SnmpPacket) (*ProbeResult, error) {
	if packet == nil {
		return nil, errors.New("SNMP response is nil")
	}

	if packet.Error != gosnmp.NoError {
		return nil, fmt.Errorf(
			"SNMP response error: status=%s index=%d",
			packet.Error.String(),
			packet.ErrorIndex,
		)
	}

	if len(packet.Variables) != 1 {
		return nil, fmt.Errorf(
			"SNMP response expected 1 variable, got %d",
			len(packet.Variables),
		)
	}

	variable := packet.Variables[0]

	switch variable.Type {
	case gosnmp.NoSuchObject:
		return nil, errors.New("SNMP OID returned noSuchObject")
	case gosnmp.NoSuchInstance:
		return nil, errors.New("SNMP OID returned noSuchInstance")
	case gosnmp.EndOfMibView:
		return nil, errors.New("SNMP OID returned endOfMibView")
	case gosnmp.Null:
		return nil, errors.New("SNMP OID returned null value")
	}

	return &ProbeResult{
		OID:   variable.Name,
		Type:  variable.Type,
		Value: variable.Value,
	}, nil
}
