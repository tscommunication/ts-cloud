package snmp

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

const (
	// Standard BRIDGE-MIB FDB. The observed SZCOM/SOLITINE OLT exposes the
	// complete customer CPE MAC here and resolves it to a bridge port.
	SZCOMDot1dTpFdbPortOID = ".1.3.6.1.2.1.17.4.3.1.2"

	// BRIDGE-MIB bridge-port -> IF-MIB ifIndex mapping.
	SZCOMDot1dBasePortIfIndexOID = ".1.3.6.1.2.1.17.1.4.1.2"
)

var szcomPONInterfaceRE = regexp.MustCompile(`(?i)^pon([0-9]+)$`)

type SZCOMLearnedMACPONResolution struct {
	MACAddress string
	PortID     int
	IfIndex    int
	Interface  string
	PONNo      int
}

type szcomFDBWalkFunc func(rootOID string) ([]WalkResult, error)
type szcomFDBGetFunc func(oid string) (*ProbeResult, error)

func ParseSZCOMPONInterface(
	name string,
) (
	ponNo int,
	ok bool,
) {
	match := szcomPONInterfaceRE.FindStringSubmatch(
		strings.TrimSpace(name),
	)

	if len(match) != 2 {
		return 0, false
	}

	ponNo, err := strconv.Atoi(match[1])
	if err != nil || ponNo <= 0 {
		return 0, false
	}

	return ponNo, true
}

func (SZCOMONUAdapter) ResolveLearnedMACPON(
	cfg V2CConfig,
	macAddress string,
) (*SZCOMLearnedMACPONResolution, error) {
	walk := func(rootOID string) ([]WalkResult, error) {
		client, err := NewV2CClient(cfg)
		if err != nil {
			return nil, err
		}

		return WalkSubtree(client, rootOID)
	}

	get := func(oid string) (*ProbeResult, error) {
		client, err := NewV2CClient(cfg)
		if err != nil {
			return nil, err
		}

		return GetOne(client, oid)
	}

	return resolveSZCOMLearnedMACPON(
		macAddress,
		walk,
		get,
	)
}

func resolveSZCOMLearnedMACPON(
	macAddress string,
	walk szcomFDBWalkFunc,
	get szcomFDBGetFunc,
) (*SZCOMLearnedMACPONResolution, error) {
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
			"SZCOM FDB walk function is required",
		)
	}

	if get == nil {
		return nil, fmt.Errorf(
			"SZCOM SNMP get function is required",
		)
	}

	rows, err := walk(SZCOMDot1dTpFdbPortOID)
	if err != nil {
		return nil, fmt.Errorf(
			"walk SZCOM bridge FDB port table: %w",
			err,
		)
	}

	portID, ok := FindVSOLLearnedMACPort(
		rows,
		parsed.String(),
	)
	if !ok {
		return nil, nil
	}

	ifIndexResult, err := get(
		fmt.Sprintf(
			"%s.%d",
			SZCOMDot1dBasePortIfIndexOID,
			portID,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get SZCOM bridge port ifIndex: %w",
			err,
		)
	}

	if ifIndexResult == nil {
		return nil, fmt.Errorf(
			"SZCOM bridge port ifIndex response is nil",
		)
	}

	ifIndex, err := IntValue(ifIndexResult.Value)
	if err != nil || ifIndex <= 0 {
		return nil, fmt.Errorf(
			"parse SZCOM bridge port ifIndex",
		)
	}

	ifNameResult, err := get(
		fmt.Sprintf("%s.%d", IFNameOID, ifIndex),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get SZCOM learned MAC interface name: %w",
			err,
		)
	}

	if ifNameResult == nil {
		return nil, fmt.Errorf(
			"SZCOM interface name response is nil",
		)
	}

	interfaceName, err := StringValue(ifNameResult.Value)
	if err != nil {
		return nil, fmt.Errorf(
			"parse SZCOM interface name: %w",
			err,
		)
	}

	ponNo, ok := ParseSZCOMPONInterface(interfaceName)
	if !ok {
		// Uplink/mgmt MACs are not customer ONU correlation evidence.
		return nil, nil
	}

	return &SZCOMLearnedMACPONResolution{
		MACAddress: strings.ToUpper(parsed.String()),
		PortID:     portID,
		IfIndex:    ifIndex,
		Interface:  interfaceName,
		PONNo:      ponNo,
	}, nil
}
