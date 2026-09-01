package snmp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	// Q-BRIDGE-MIB::dot1qTpFdbPort.
	//
	// The observed BDCOM OLT exposes learned customer CPE MAC entries as:
	//
	//   <root>.<vlan>.<mac-octet-1>...<mac-octet-6> = <bridge-port>
	//
	// Example:
	//
	//   ...1.2.624.80.210.245.184.220.78 = 284
	//
	// where VLAN=624, MAC=50:D2:F5:B8:DC:4E and bridge port 284 is
	// also exposed through IF-MIB as EPON0/4:8.
	BDCOMDot1qTpFdbPortOID = ".1.3.6.1.2.1.17.7.1.2.2.1.2"
)

type BDCOMLearnedMACResolution struct {
	MACAddress string
	VLAN       int
	PortID     int
	Interface  string
	PONNo      int
	ONUNo      int
}

func parseBDCOMQBridgeFDBPortOID(
	oid string,
	rootOID string,
) (
	vlan int,
	macAddress string,
	ok bool,
) {
	root := strings.TrimPrefix(
		strings.TrimSpace(rootOID),
		".",
	)

	value := strings.TrimPrefix(
		strings.TrimSpace(oid),
		".",
	)

	prefix := root + "."
	if !strings.HasPrefix(value, prefix) {
		return 0, "", false
	}

	suffix := strings.TrimPrefix(value, prefix)
	parts := strings.Split(suffix, ".")

	// Q-BRIDGE dot1qTpFdbPort index:
	// VLAN ID + six MAC address octets.
	if len(parts) != 7 {
		return 0, "", false
	}

	vlan, err := strconv.Atoi(parts[0])
	if err != nil || vlan <= 0 {
		return 0, "", false
	}

	octets := make([]byte, 6)

	for i := 0; i < 6; i++ {
		n, err := strconv.Atoi(parts[i+1])
		if err != nil || n < 0 || n > 255 {
			return 0, "", false
		}

		octets[i] = byte(n)
	}

	mac := net.HardwareAddr(octets)

	return vlan, strings.ToUpper(mac.String()), true
}

func BDCOMFindLearnedMACPort(
	rows []WalkResult,
	macAddress string,
) (
	vlan int,
	portID int,
	ok bool,
) {
	parsed, err := net.ParseMAC(
		strings.TrimSpace(macAddress),
	)
	if err != nil {
		return 0, 0, false
	}

	target := strings.ToUpper(parsed.String())

	for _, row := range rows {
		rowVLAN, rowMAC, parsedOK :=
			parseBDCOMQBridgeFDBPortOID(
				row.OID,
				BDCOMDot1qTpFdbPortOID,
			)

		if !parsedOK || rowMAC != target {
			continue
		}

		port, valueOK := integerSNMPValue(row.Value)
		if !valueOK || port <= 0 {
			continue
		}

		return rowVLAN, port, true
	}

	return 0, 0, false
}

func integerSNMPValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		if typed > uint64(^uint(0)>>1) {
			return 0, false
		}
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(
			strings.TrimSpace(typed),
		)
		return parsed, err == nil
	default:
		parsed, err := strconv.Atoi(
			strings.TrimSpace(fmt.Sprint(value)),
		)
		return parsed, err == nil
	}
}

type bdcomFDBWalkFunc func(
	rootOID string,
) ([]WalkResult, error)

type bdcomFDBGetFunc func(
	oid string,
) (*ProbeResult, error)

func (BDCOMONUAdapter) ResolveLearnedMAC(
	cfg V2CConfig,
	macAddress string,
) (*BDCOMLearnedMACResolution, error) {
	walk := func(
		rootOID string,
	) ([]WalkResult, error) {
		client, err := NewV2CClient(cfg)
		if err != nil {
			return nil, err
		}

		return WalkSubtree(client, rootOID)
	}

	get := func(
		oid string,
	) (*ProbeResult, error) {
		client, err := NewV2CClient(cfg)
		if err != nil {
			return nil, err
		}

		return GetOne(client, oid)
	}

	return resolveBDCOMLearnedMAC(
		macAddress,
		walk,
		get,
	)
}

func resolveBDCOMLearnedMAC(
	macAddress string,
	walk bdcomFDBWalkFunc,
	get bdcomFDBGetFunc,
) (*BDCOMLearnedMACResolution, error) {
	parsed, err := net.ParseMAC(
		strings.TrimSpace(macAddress),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid customer MAC address: %w",
			err,
		)
	}

	if walk == nil {
		return nil, fmt.Errorf(
			"BDCOM FDB walk function is required",
		)
	}

	if get == nil {
		return nil, fmt.Errorf(
			"BDCOM interface get function is required",
		)
	}

	rows, err := walk(BDCOMDot1qTpFdbPortOID)
	if err != nil {
		return nil, fmt.Errorf(
			"walk BDCOM Q-BRIDGE FDB port table: %w",
			err,
		)
	}

	vlan, portID, ok := BDCOMFindLearnedMACPort(
		rows,
		parsed.String(),
	)
	if !ok {
		return nil, nil
	}

	// Live BDCOM hardware exposes the Q-BRIDGE learned port as the
	// same IF-MIB ifIndex. Validate the interface name before using
	// the position instead of trusting the integer blindly.
	result, err := get(
		fmt.Sprintf("%s.%d", IFNameOID, portID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get BDCOM learned MAC interface name: %w",
			err,
		)
	}

	if result == nil {
		return nil, fmt.Errorf(
			"BDCOM learned MAC interface response is nil",
		)
	}

	interfaceName, err := StringValue(result.Value)
	if err != nil {
		return nil, fmt.Errorf(
			"parse BDCOM learned MAC interface name: %w",
			err,
		)
	}

	ponNo, onuNo, ok :=
		parseBDCOMONUIfDescr(interfaceName)

	if !ok {
		return nil, fmt.Errorf(
			"BDCOM learned MAC port %d resolved to unsupported interface %q",
			portID,
			interfaceName,
		)
	}

	return &BDCOMLearnedMACResolution{
		MACAddress: strings.ToUpper(parsed.String()),
		VLAN:       vlan,
		PortID:     portID,
		Interface:  interfaceName,
		PONNo:      ponNo,
		ONUNo:      onuNo,
	}, nil
}
