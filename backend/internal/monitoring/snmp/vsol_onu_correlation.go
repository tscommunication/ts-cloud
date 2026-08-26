package snmp

import (
	"fmt"
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
