package snmp

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	vsolIFMIBLegacyONUNameRE = regexp.MustCompile(
		`(?i)^EPON([0-9]{2})ONU([0-9]+)(?:\s|$)`,
	)

	vsolIFMIBSlashONUNameRE = regexp.MustCompile(
		`(?i)^EPON0/([0-9]+):([0-9]+)(?:\s|$)`,
	)
)

type vsolIFMIBONUKey struct {
	PONNo int
	ONUNo int
}

func ParseVSOLIFMIBONUName(
	name string,
) (
	ponNo int,
	onuNo int,
	ok bool,
) {
	trimmed := strings.TrimSpace(name)

	match :=
		vsolIFMIBLegacyONUNameRE.FindStringSubmatch(
			trimmed,
		)

	if len(match) != 3 {
		match =
			vsolIFMIBSlashONUNameRE.FindStringSubmatch(
				trimmed,
			)
	}

	if len(match) != 3 {
		return 0, 0, false
	}

	ponNo, err1 := strconv.Atoi(match[1])
	onuNo, err2 := strconv.Atoi(match[2])

	if err1 != nil ||
		err2 != nil ||
		ponNo <= 0 ||
		onuNo <= 0 {
		return 0, 0, false
	}

	return ponNo, onuNo, true
}

const (
	VSOLDot1dTpFdbPortOID = ".1.3.6.1.2.1.17.4.3.1.2"
)

type VSOLLearnedMACResolution struct {
	MACAddress string
	PortID     int
	Interface  string
	PONNo      int
	ONUNo      int
}

func parseVSOLFDBPortOID(
	oid string,
	rootOID string,
) (string, bool) {
	root := strings.TrimSuffix(strings.TrimSpace(rootOID), ".")
	value := strings.TrimSpace(oid)
	prefix := root + "."

	if !strings.HasPrefix(value, prefix) {
		return "", false
	}

	suffix := strings.TrimPrefix(value, prefix)
	parts := strings.Split(suffix, ".")

	// Most BRIDGE-MIB agents use six MAC octets as the index.
	// The production VSOL OLT currently exposes:
	//
	//   <length=6>.<octet1>...<octet6>
	//
	// Accept both shapes so the parser remains standards-friendly while
	// preserving compatibility with the observed VSOL agent.
	if len(parts) == 7 && parts[0] == "6" {
		parts = parts[1:]
	}

	if len(parts) != 6 {
		return "", false
	}

	raw := make([]byte, 6)

	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 || value > 255 {
			return "", false
		}
		raw[i] = byte(value)
	}

	return strings.ToUpper(net.HardwareAddr(raw).String()), true
}

func FindVSOLLearnedMACPort(
	rows []WalkResult,
	macAddress string,
) (int, bool) {
	parsed, err := net.ParseMAC(strings.TrimSpace(macAddress))
	if err != nil {
		return 0, false
	}

	target := strings.ToUpper(parsed.String())

	for _, row := range rows {
		mac, ok := parseVSOLFDBPortOID(
			row.OID,
			VSOLDot1dTpFdbPortOID,
		)
		if !ok || !strings.EqualFold(mac, target) {
			continue
		}

		portID, err := IntValue(row.Value)
		if err != nil || portID <= 0 {
			return 0, false
		}

		return portID, true
	}

	return 0, false
}

type vsolFDBWalkFunc func(rootOID string) ([]WalkResult, error)
type vsolFDBGetFunc func(oid string) (*ProbeResult, error)

func (VSOLONUAdapter) ResolveLearnedMAC(
	cfg V2CConfig,
	macAddress string,
) (*VSOLLearnedMACResolution, error) {
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

	return resolveVSOLLearnedMAC(macAddress, walk, get)
}

func resolveVSOLLearnedMAC(
	macAddress string,
	walk vsolFDBWalkFunc,
	get vsolFDBGetFunc,
) (*VSOLLearnedMACResolution, error) {
	parsed, err := net.ParseMAC(strings.TrimSpace(macAddress))
	if err != nil {
		return nil, fmt.Errorf(
			"invalid customer MAC address: %w",
			err,
		)
	}

	if walk == nil {
		return nil, fmt.Errorf(
			"VSOL FDB walk function is required",
		)
	}

	if get == nil {
		return nil, fmt.Errorf(
			"VSOL interface get function is required",
		)
	}

	rows, err := walk(VSOLDot1dTpFdbPortOID)
	if err != nil {
		return nil, fmt.Errorf(
			"walk VSOL bridge FDB port table: %w",
			err,
		)
	}

	portID, ok := FindVSOLLearnedMACPort(rows, parsed.String())
	if !ok {
		return nil, nil
	}

	// On the observed VSOL OLT the learned bridge port is the same index
	// exposed by IF-MIB/IF-MIB::ifName. Validate that relationship instead
	// of blindly accepting the integer as an ONU position.
	result, err := get(
		fmt.Sprintf("%s.%d", IFNameOID, portID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get VSOL learned MAC interface name: %w",
			err,
		)
	}

	if result == nil {
		return nil, fmt.Errorf(
			"VSOL learned MAC interface response is nil",
		)
	}

	interfaceName, err := StringValue(result.Value)
	if err != nil {
		return nil, fmt.Errorf(
			"parse VSOL learned MAC interface name: %w",
			err,
		)
	}

	ponNo, onuNo, ok :=
		ParseVSOLIFMIBONUName(interfaceName)

	if !ok {
		return nil, fmt.Errorf(
			"VSOL learned MAC port %d resolved to unsupported interface %q",
			portID,
			interfaceName,
		)
	}

	return &VSOLLearnedMACResolution{
		MACAddress: strings.ToUpper(parsed.String()),
		PortID:     portID,
		Interface:  interfaceName,
		PONNo:      ponNo,
		ONUNo:      onuNo,
	}, nil
}

func BuildVSOLONUPersistenceCandidates(
	ifmib *IFMIBCollection,
	optical *ONUOpticalCollection,
) ([]ONUPersistenceCandidate, error) {
	var sampledAt time.Time

	if optical != nil {
		if optical.SampledAt.IsZero() {
			return nil, fmt.Errorf(
				"VSOL ONU optical sample time is required",
			)
		}

		if len(optical.Records) == 0 {
			return nil, fmt.Errorf(
				"VSOL ONU optical records are required",
			)
		}

		sampledAt = optical.SampledAt
	} else {
		if ifmib == nil {
			return nil, fmt.Errorf(
				"VSOL ONU IF-MIB collection is required when optical telemetry is unavailable",
			)
		}

		if ifmib.SampledAt.IsZero() {
			return nil, fmt.Errorf(
				"VSOL ONU IF-MIB sample time is required",
			)
		}

		sampledAt = ifmib.SampledAt
	}

	ifPorts := make(
		map[vsolIFMIBONUKey]IFMIBPort,
	)

	if ifmib != nil {
		for _, port := range ifmib.Ports {
			name := strings.TrimSpace(port.Name)

			ponNo, onuNo, ok :=
				ParseVSOLIFMIBONUName(name)

			if !ok {
				continue
			}

			key := vsolIFMIBONUKey{
				PONNo: ponNo,
				ONUNo: onuNo,
			}

			if _, exists := ifPorts[key]; exists {
				return nil, fmt.Errorf(
					"duplicate VSOL IF-MIB ONU mapping PON=%d ONU=%d",
					ponNo,
					onuNo,
				)
			}

			ifPorts[key] = port
		}
	}

	if optical == nil {
		if len(ifPorts) == 0 {
			return nil, fmt.Errorf(
				"VSOL ONU IF-MIB collection contained no ONU interfaces",
			)
		}

		keys := make(
			[]vsolIFMIBONUKey,
			0,
			len(ifPorts),
		)

		for key := range ifPorts {
			keys = append(keys, key)
		}

		sort.Slice(keys, func(i, j int) bool {
			if keys[i].PONNo != keys[j].PONNo {
				return keys[i].PONNo < keys[j].PONNo
			}

			return keys[i].ONUNo < keys[j].ONUNo
		})

		candidates := make(
			[]ONUPersistenceCandidate,
			0,
			len(keys),
		)

		for _, key := range keys {
			port := ifPorts[key]

			candidates = append(
				candidates,
				ONUPersistenceCandidate{
					PONNo:       key.PONNo,
					ONUNo:       key.ONUNo,
					IfIndex:     port.IfIndex,
					Description: strings.TrimSpace(port.Name),
					OperStatus:  InterfaceStatus(port.OperStatus),
					MACAddress:  strings.TrimSpace(port.MACAddress),
					InOctets:    CounterValue(port.HCInOctets),
					OutOctets:   CounterValue(port.HCOutOctets),
					SampledAt:   sampledAt,
				},
			)
		}

		return candidates, nil
	}

	seenOptical := make(
		map[vsolIFMIBONUKey]struct{},
	)

	candidates := make(
		[]ONUPersistenceCandidate,
		0,
		len(optical.Records),
	)

	for _, record := range optical.Records {
		if record.PONNo <= 0 ||
			record.ONUNo <= 0 {
			return nil, fmt.Errorf(
				"invalid VSOL ONU key PON=%d ONU=%d",
				record.PONNo,
				record.ONUNo,
			)
		}

		key := vsolIFMIBONUKey{
			PONNo: record.PONNo,
			ONUNo: record.ONUNo,
		}

		if _, exists := seenOptical[key]; exists {
			return nil, fmt.Errorf(
				"duplicate VSOL optical ONU PON=%d ONU=%d",
				record.PONNo,
				record.ONUNo,
			)
		}

		seenOptical[key] = struct{}{}

		candidate := ONUPersistenceCandidate{
			PONNo:        record.PONNo,
			ONUNo:        record.ONUNo,
			OperStatus:   "UNKNOWN",
			TemperatureC: record.TemperatureC,
			VoltageV:     record.VoltageV,
			TxBiasMA:     record.TxBiasMA,
			TxPowerDBM:   record.TxPowerDBM,
			RxPowerDBM:   record.RxPowerDBM,
			SampledAt:    optical.SampledAt,
		}

		if port, exists := ifPorts[key]; exists {
			candidate.IfIndex = port.IfIndex
			candidate.Description =
				strings.TrimSpace(port.Name)
			candidate.OperStatus =
				InterfaceStatus(port.OperStatus)
			candidate.MACAddress =
				strings.TrimSpace(port.MACAddress)
			candidate.InOctets =
				CounterValue(port.HCInOctets)
			candidate.OutOctets =
				CounterValue(port.HCOutOctets)
		}

		candidates = append(
			candidates,
			candidate,
		)
	}

	return candidates, nil
}

func MergeVSOLONURegistrationTimes(
	candidates []ONUPersistenceCandidate,
	records []VSOLONURegistrationRecord,
) []ONUPersistenceCandidate {
	if len(candidates) == 0 ||
		len(records) == 0 {
		return candidates
	}

	byKey := make(
		map[vsolONUKey]VSOLONURegistrationRecord,
		len(records),
	)

	for _, record := range records {
		if record.PONNo <= 0 ||
			record.ONUNo <= 0 {
			continue
		}

		byKey[vsolONUKey{
			PONNo: record.PONNo,
			ONUNo: record.ONUNo,
		}] = record
	}

	for index := range candidates {
		key := vsolONUKey{
			PONNo: candidates[index].PONNo,
			ONUNo: candidates[index].ONUNo,
		}

		record, ok := byKey[key]
		if !ok {
			continue
		}

		candidates[index].LastRegisteredAt =
			record.LastRegisteredAt

		candidates[index].LastDeregisteredAt =
			record.LastDeregisteredAt
	}

	return candidates
}
