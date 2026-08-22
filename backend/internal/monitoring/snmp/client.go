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

type TransportError struct {
	Operation string
	Err       error
}

func (e *TransportError) Error() string {
	if e == nil {
		return "SNMP transport error"
	}
	if e.Err == nil {
		return fmt.Sprintf("SNMP %s failed", e.Operation)
	}
	return fmt.Sprintf("SNMP %s: %v", e.Operation, e.Err)
}

func (e *TransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ResponseErrorKind string

const (
	ResponseNil            ResponseErrorKind = "nil_response"
	ResponseProtocolError  ResponseErrorKind = "protocol_error"
	ResponseVariableCount  ResponseErrorKind = "variable_count"
	ResponseNoSuchObject   ResponseErrorKind = "no_such_object"
	ResponseNoSuchInstance ResponseErrorKind = "no_such_instance"
	ResponseEndOfMibView   ResponseErrorKind = "end_of_mib_view"
	ResponseNullValue      ResponseErrorKind = "null_value"
)

type ResponseError struct {
	Kind   ResponseErrorKind
	Detail string
}

func (e *ResponseError) Error() string {
	if e == nil {
		return "SNMP response error"
	}
	if strings.TrimSpace(e.Detail) == "" {
		return fmt.Sprintf("SNMP response error: %s", e.Kind)
	}
	return fmt.Sprintf("SNMP response error: %s: %s", e.Kind, e.Detail)
}

func IsTransportError(err error) bool {
	var target *TransportError
	return errors.As(err, &target)
}

func IsResponseError(err error) bool {
	var target *ResponseError
	return errors.As(err, &target)
}

func ResponseErrorKindOf(err error) (ResponseErrorKind, bool) {
	var target *ResponseError
	if !errors.As(err, &target) || target == nil {
		return "", false
	}
	return target.Kind, true
}

func NewV2CClient(cfg V2CConfig) (*gosnmp.GoSNMP, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return nil, errors.New("SNMP host is required")
	}

	if strings.TrimSpace(cfg.Community) == "" {
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
		Community: cfg.Community,
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
		return nil, &TransportError{
			Operation: "connect",
			Err:       err,
		}
	}
	defer client.Conn.Close()

	packet, err := client.Get([]string{oid})
	if err != nil {
		return nil, &TransportError{
			Operation: "GET",
			Err:       err,
		}
	}

	return ValidateSingleVarBind(packet)
}

func ValidateSingleVarBind(packet *gosnmp.SnmpPacket) (*ProbeResult, error) {
	if packet == nil {
		return nil, &ResponseError{
			Kind:   ResponseNil,
			Detail: "response is nil",
		}
	}

	if packet.Error != gosnmp.NoError {
		return nil, &ResponseError{
			Kind: ResponseProtocolError,
			Detail: fmt.Sprintf(
				"status=%s index=%d",
				packet.Error.String(),
				packet.ErrorIndex,
			),
		}
	}

	if len(packet.Variables) != 1 {
		return nil, &ResponseError{
			Kind: ResponseVariableCount,
			Detail: fmt.Sprintf(
				"expected 1 variable, got %d",
				len(packet.Variables),
			),
		}
	}

	variable := packet.Variables[0]

	switch variable.Type {
	case gosnmp.NoSuchObject:
		return nil, &ResponseError{
			Kind:   ResponseNoSuchObject,
			Detail: "OID returned noSuchObject",
		}
	case gosnmp.NoSuchInstance:
		return nil, &ResponseError{
			Kind:   ResponseNoSuchInstance,
			Detail: "OID returned noSuchInstance",
		}
	case gosnmp.EndOfMibView:
		return nil, &ResponseError{
			Kind:   ResponseEndOfMibView,
			Detail: "OID returned endOfMibView",
		}
	case gosnmp.Null:
		return nil, &ResponseError{
			Kind:   ResponseNullValue,
			Detail: "OID returned null value",
		}
	}

	return &ProbeResult{
		OID:   variable.Name,
		Type:  variable.Type,
		Value: variable.Value,
	}, nil
}
