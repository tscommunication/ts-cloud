package snmp

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	VSOLEnterpriseOID = ".1.3.6.1.4.1.37950"

	VSOLONUOpticalRootOID = ".1.3.6.1.4.1.37950.1.1.5.12.2.1.8.1"

	VSOLONULastRegisteredOID = ".1.3.6.1.4.1.37950.1.1.5.12.1.25.1.18"

	VSOLONULastDeregisteredOID = ".1.3.6.1.4.1.37950.1.1.5.12.1.25.1.19"
)

type VSOLONUAdapter struct{}

func (VSOLONUAdapter) Name() string {
	return "VSOL"
}

func (VSOLONUAdapter) Matches(
	vendor string,
	sysObjectID string,
) bool {
	switch normalizedVendor(vendor) {
	case "VSOL", "ZIBBIX":
		return true
	}

	oid := strings.TrimSpace(sysObjectID)

	return oid == VSOLEnterpriseOID ||
		strings.HasPrefix(
			oid,
			VSOLEnterpriseOID+".",
		)
}

func (VSOLONUAdapter) CollectOptical(
	cfg V2CConfig,
	sampledAt time.Time,
) (*ONUOpticalCollection, error) {
	if sampledAt.IsZero() {
		return nil, fmt.Errorf("sample time is required")
	}

	client, err := NewV2CClient(cfg)
	if err != nil {
		return nil, err
	}

	rows, err := WalkSubtree(
		client,
		VSOLONUOpticalRootOID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"walk VSOL ONU optical subtree: %w",
			err,
		)
	}

	records, err := ParseVSOLONUOpticalRows(rows)
	if err != nil {
		return nil, err
	}

	return &ONUOpticalCollection{
		Vendor:    "VSOL",
		SampledAt: sampledAt,
		Records:   records,
	}, nil
}

type VSOLONURegistrationRecord struct {
	PONNo              int
	ONUNo              int
	LastRegisteredAt   *time.Time
	LastDeregisteredAt *time.Time
}

func CollectVSOLONURegistrationTimes(
	cfg V2CConfig,
	location *time.Location,
) ([]VSOLONURegistrationRecord, error) {
	if location == nil {
		return nil, fmt.Errorf("VSOL ONU timestamp location is required")
	}

	records := make(
		map[vsolONUKey]VSOLONURegistrationRecord,
	)

	collect := func(
		rootOID string,
		assign func(
			*VSOLONURegistrationRecord,
			*time.Time,
		),
	) error {
		client, err := NewV2CClient(cfg)
		if err != nil {
			return err
		}

		rows, err := WalkSubtree(
			client,
			rootOID,
		)
		if err != nil {
			return err
		}

		for _, row := range rows {
			ponNo, onuNo, ok :=
				parseVSOLONUPONONUOID(
					row.OID,
					rootOID,
				)

			if !ok {
				continue
			}

			parsed := parseVSOLONUTimestamp(
				walkResultText(row.Value),
				location,
			)

			key := vsolONUKey{
				PONNo: ponNo,
				ONUNo: onuNo,
			}

			record := records[key]

			record.PONNo = ponNo
			record.ONUNo = onuNo

			assign(
				&record,
				parsed,
			)

			records[key] = record
		}

		return nil
	}

	if err := collect(
		VSOLONULastRegisteredOID,
		func(
			record *VSOLONURegistrationRecord,
			value *time.Time,
		) {
			record.LastRegisteredAt = value
		},
	); err != nil {
		return nil, fmt.Errorf(
			"walk VSOL ONU last registered timestamps: %w",
			err,
		)
	}

	if err := collect(
		VSOLONULastDeregisteredOID,
		func(
			record *VSOLONURegistrationRecord,
			value *time.Time,
		) {
			record.LastDeregisteredAt = value
		},
	); err != nil {
		return nil, fmt.Errorf(
			"walk VSOL ONU last deregistered timestamps: %w",
			err,
		)
	}

	keys := make(
		[]vsolONUKey,
		0,
		len(records),
	)

	for key := range records {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].PONNo != keys[j].PONNo {
			return keys[i].PONNo < keys[j].PONNo
		}

		return keys[i].ONUNo < keys[j].ONUNo
	})

	result := make(
		[]VSOLONURegistrationRecord,
		0,
		len(keys),
	)

	for _, key := range keys {
		result = append(
			result,
			records[key],
		)
	}

	return result, nil
}

func (VSOLONUAdapter) CollectRegistrationTimes(
	cfg V2CConfig,
	location *time.Location,
) ([]VSOLONURegistrationRecord, error) {
	return CollectVSOLONURegistrationTimes(
		cfg,
		location,
	)
}

func parseVSOLONUPONONUOID(
	oid string,
	rootOID string,
) (
	ponNo int,
	onuNo int,
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
		return 0, 0, false
	}

	suffix := strings.Split(
		strings.TrimPrefix(value, prefix),
		".",
	)

	if len(suffix) != 2 {
		return 0, 0, false
	}

	ponNo, err1 := strconv.Atoi(suffix[0])
	onuNo, err2 := strconv.Atoi(suffix[1])

	if err1 != nil ||
		err2 != nil ||
		ponNo <= 0 ||
		onuNo <= 0 {
		return 0, 0, false
	}

	return ponNo, onuNo, true
}

func parseVSOLONUTimestamp(
	value string,
	location *time.Location,
) *time.Time {
	value = strings.TrimSpace(value)

	if value == "" ||
		strings.EqualFold(value, "N/A") {
		return nil
	}

	parsed, err := time.ParseInLocation(
		"2006/01/02 15:04:05",
		value,
		location,
	)
	if err != nil {
		return nil
	}

	return &parsed
}

type vsolONUKey struct {
	PONNo int
	ONUNo int
}

var vsolNumericValueRE = regexp.MustCompile(
	`^\s*(-?\d+(?:\.\d+)?)`,
)

var vsolDBMValueRE = regexp.MustCompile(
	`\((-?\d+(?:\.\d+)?)\s*dBm\)`,
)

func ParseVSOLONUOpticalRows(
	rows []WalkResult,
) ([]ONUOpticalRecord, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf(
			"VSOL ONU optical rows are required",
		)
	}

	records := make(
		map[vsolONUKey]*ONUOpticalRecord,
	)

	for _, row := range rows {
		column, ponNo, onuNo, ok :=
			parseVSOLONUOpticalOID(row.OID)

		if !ok {
			continue
		}

		key := vsolONUKey{
			PONNo: ponNo,
			ONUNo: onuNo,
		}

		record := records[key]
		if record == nil {
			record = &ONUOpticalRecord{
				PONNo: ponNo,
				ONUNo: onuNo,
			}
			records[key] = record
		}

		value := walkResultText(row.Value)

		switch column {
		case 1:
			// PON number is also encoded in the OID suffix.
		case 2:
			// ONU number is also encoded in the OID suffix.
		case 3:
			record.TemperatureC =
				parseVSOLNumericPointer(value)
		case 4:
			record.VoltageV =
				parseVSOLNumericPointer(value)
		case 5:
			record.TxBiasMA =
				parseVSOLNumericPointer(value)
		case 6:
			record.TxPowerDBM =
				parseVSOLDBMPointer(value)
		case 7:
			record.RxPowerDBM =
				parseVSOLDBMPointer(value)
		}
	}

	if len(records) == 0 {
		return nil, fmt.Errorf(
			"VSOL ONU optical subtree contained no usable rows",
		)
	}

	keys := make([]vsolONUKey, 0, len(records))

	for key := range records {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].PONNo != keys[j].PONNo {
			return keys[i].PONNo < keys[j].PONNo
		}

		return keys[i].ONUNo < keys[j].ONUNo
	})

	result := make(
		[]ONUOpticalRecord,
		0,
		len(keys),
	)

	for _, key := range keys {
		result = append(
			result,
			*records[key],
		)
	}

	return result, nil
}

func parseVSOLONUOpticalOID(
	oid string,
) (
	column int,
	ponNo int,
	onuNo int,
	ok bool,
) {
	root := strings.TrimPrefix(
		VSOLONUOpticalRootOID,
		".",
	)

	value := strings.TrimPrefix(
		strings.TrimSpace(oid),
		".",
	)

	prefix := root + "."

	if !strings.HasPrefix(value, prefix) {
		return 0, 0, 0, false
	}

	suffix := strings.Split(
		strings.TrimPrefix(value, prefix),
		".",
	)

	if len(suffix) != 3 {
		return 0, 0, 0, false
	}

	column, err1 := strconv.Atoi(suffix[0])
	ponNo, err2 := strconv.Atoi(suffix[1])
	onuNo, err3 := strconv.Atoi(suffix[2])

	if err1 != nil ||
		err2 != nil ||
		err3 != nil ||
		column < 1 ||
		column > 7 ||
		ponNo <= 0 ||
		onuNo <= 0 {
		return 0, 0, 0, false
	}

	return column, ponNo, onuNo, true
}

func walkResultText(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return strings.TrimSpace(
			fmt.Sprint(value),
		)
	}
}

func parseVSOLNumericPointer(
	value string,
) *float64 {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	match := vsolNumericValueRE.FindStringSubmatch(
		value,
	)

	if len(match) != 2 {
		return nil
	}

	number, err := strconv.ParseFloat(
		match[1],
		64,
	)
	if err != nil {
		return nil
	}

	return &number
}

func parseVSOLDBMPointer(
	value string,
) *float64 {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	match := vsolDBMValueRE.FindStringSubmatch(
		value,
	)

	if len(match) != 2 {
		return nil
	}

	number, err := strconv.ParseFloat(
		match[1],
		64,
	)
	if err != nil {
		return nil
	}

	return &number
}

func (VSOLONUAdapter) BuildPersistenceCandidates(
	ifmib *IFMIBCollection,
	optical *ONUOpticalCollection,
) ([]ONUPersistenceCandidate, error) {
	return BuildVSOLONUPersistenceCandidates(
		ifmib,
		optical,
	)
}
