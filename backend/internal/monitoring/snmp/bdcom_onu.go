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
	BDCOMEnterpriseOID = ".1.3.6.1.4.1.3320"

	BDCOMONUVendorOID         = ".1.3.6.1.4.1.3320.101.10.1.1.1"
	BDCOMONUModelOID          = ".1.3.6.1.4.1.3320.101.10.1.1.2"
	BDCOMONUStatusOID         = ".1.3.6.1.4.1.3320.101.10.1.1.26"
	BDCOMONUMACOID            = ".1.3.6.1.4.1.3320.101.10.4.1.1"
	BDCOMONUOpticalRxPowerOID = ".1.3.6.1.4.1.3320.101.10.5.1.5"
	BDCOMONUOpticalTxPowerOID = ".1.3.6.1.4.1.3320.101.10.5.1.6"
)

type BDCOMONUAdapter struct{}

func (BDCOMONUAdapter) Name() string {
	return "BDCOM"
}

func (BDCOMONUAdapter) Matches(
	vendor string,
	sysObjectID string,
) bool {
	if normalizedVendor(vendor) == "BDCOM" {
		return true
	}

	oid := strings.TrimSpace(sysObjectID)

	return oid == BDCOMEnterpriseOID ||
		strings.HasPrefix(
			oid,
			BDCOMEnterpriseOID+".",
		)
}

type BDCOMONUInventoryRecord struct {
	IfIndex    int
	PONNo      int
	ONUNo      int
	Vendor     string
	Model      string
	MACAddress string
	OperStatus string
}

type BDCOMONUInventoryCollection struct {
	Vendor    string
	SampledAt time.Time
	Records   []BDCOMONUInventoryRecord
}

type BDCOMONUInventoryCollector interface {
	CollectInventory(
		cfg V2CConfig,
		sampledAt time.Time,
	) (*BDCOMONUInventoryCollection, error)
}

func (BDCOMONUAdapter) CollectInventory(
	cfg V2CConfig,
	sampledAt time.Time,
) (*BDCOMONUInventoryCollection, error) {
	if sampledAt.IsZero() {
		return nil, fmt.Errorf(
			"sample time is required",
		)
	}

	client, err := NewV2CClient(cfg)
	if err != nil {
		return nil, err
	}

	records := make(
		map[int]*BDCOMONUInventoryRecord,
	)

	get := func(ifIndex int) *BDCOMONUInventoryRecord {
		record := records[ifIndex]

		if record == nil {
			record = &BDCOMONUInventoryRecord{
				IfIndex: ifIndex,
			}
			records[ifIndex] = record
		}

		return record
	}

	walkColumn := func(
		rootOID string,
		assign func(
			*BDCOMONUInventoryRecord,
			any,
		),
	) error {
		rows, err := WalkSubtree(
			client,
			rootOID,
		)
		if err != nil {
			return err
		}

		for _, row := range rows {
			ifIndex, ok := parseBDCOMSingleIndexOID(
				row.OID,
				rootOID,
			)

			if !ok {
				continue
			}

			assign(
				get(ifIndex),
				row.Value,
			)
		}

		return nil
	}

	if err := walkColumn(
		BDCOMONUVendorOID,
		func(
			record *BDCOMONUInventoryRecord,
			value any,
		) {
			record.Vendor = strings.TrimSpace(
				walkResultText(value),
			)
		},
	); err != nil {
		return nil, fmt.Errorf(
			"walk BDCOM ONU vendor table: %w",
			err,
		)
	}

	if err := walkColumn(
		BDCOMONUModelOID,
		func(
			record *BDCOMONUInventoryRecord,
			value any,
		) {
			record.Model = strings.TrimSpace(
				walkResultText(value),
			)
		},
	); err != nil {
		return nil, fmt.Errorf(
			"walk BDCOM ONU model table: %w",
			err,
		)
	}

	if err := walkColumn(
		BDCOMONUStatusOID,
		func(
			record *BDCOMONUInventoryRecord,
			value any,
		) {
			record.OperStatus =
				parseBDCOMONUStatus(
					walkResultText(value),
				)
		},
	); err != nil {
		return nil, fmt.Errorf(
			"walk BDCOM ONU status table: %w",
			err,
		)
	}

	if err := walkColumn(
		BDCOMONUMACOID,
		func(
			record *BDCOMONUInventoryRecord,
			value any,
		) {
			record.MACAddress =
				parseBDCOMMACValue(value)
		},
	); err != nil {
		return nil, fmt.Errorf(
			"walk BDCOM ONU MAC table: %w",
			err,
		)
	}

	keys := make([]int, 0, len(records))

	for ifIndex := range records {
		keys = append(keys, ifIndex)
	}

	sort.Ints(keys)

	result := make(
		[]BDCOMONUInventoryRecord,
		0,
		len(keys),
	)

	for _, ifIndex := range keys {
		result = append(
			result,
			*records[ifIndex],
		)
	}

	return &BDCOMONUInventoryCollection{
		Vendor:    "BDCOM",
		SampledAt: sampledAt,
		Records:   result,
	}, nil
}

func MergeBDCOMONUInventory(
	candidates []ONUPersistenceCandidate,
	inventory *BDCOMONUInventoryCollection,
) []ONUPersistenceCandidate {
	if len(candidates) == 0 ||
		inventory == nil ||
		len(inventory.Records) == 0 {
		return candidates
	}

	byIfIndex := make(
		map[int]BDCOMONUInventoryRecord,
		len(inventory.Records),
	)

	for _, record := range inventory.Records {
		if record.IfIndex <= 0 {
			continue
		}

		byIfIndex[record.IfIndex] = record
	}

	for index := range candidates {
		record, ok := byIfIndex[candidates[index].IfIndex]

		if !ok {
			continue
		}

		if value := strings.TrimSpace(
			record.MACAddress,
		); value != "" {
			candidates[index].MACAddress = value
		}

		if value := strings.TrimSpace(
			record.Model,
		); value != "" &&
			value != "----" {
			candidates[index].Model = value
		}

		if record.OperStatus == "UP" ||
			record.OperStatus == "DOWN" {
			candidates[index].OperStatus =
				record.OperStatus
		}
	}

	return candidates
}

var bdcomONUIfDescrRE = regexp.MustCompile(
	`(?i)^EPON\d+/(\d+):(\d+)$`,
)

func parseBDCOMONUIfDescr(
	value string,
) (
	ponNo int,
	onuNo int,
	ok bool,
) {
	match := bdcomONUIfDescrRE.FindStringSubmatch(
		strings.TrimSpace(value),
	)

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

func parseBDCOMONUStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "3":
		return "UP"
	case "2":
		return "DOWN"
	default:
		return "UNKNOWN"
	}
}

func parseBDCOMSingleIndexOID(
	oid string,
	rootOID string,
) (
	int,
	bool,
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
		return 0, false
	}

	suffix := strings.TrimPrefix(value, prefix)

	if strings.Contains(suffix, ".") {
		return 0, false
	}

	index, err := strconv.Atoi(suffix)

	if err != nil || index <= 0 {
		return 0, false
	}

	return index, true
}

func parseBDCOMMACValue(value any) string {
	switch typed := value.(type) {
	case []byte:
		if len(typed) == 6 {
			return fmt.Sprintf(
				"%02X:%02X:%02X:%02X:%02X:%02X",
				typed[0],
				typed[1],
				typed[2],
				typed[3],
				typed[4],
				typed[5],
			)
		}
	}

	return strings.TrimSpace(
		walkResultText(value),
	)
}

func (BDCOMONUAdapter) CollectOptical(
	cfg V2CConfig,
	sampledAt time.Time,
) (*ONUOpticalCollection, error) {
	if sampledAt.IsZero() {
		return nil, fmt.Errorf(
			"sample time is required",
		)
	}

	client, err := NewV2CClient(cfg)
	if err != nil {
		return nil, err
	}

	rxRows, err := WalkSubtree(
		client,
		BDCOMONUOpticalRxPowerOID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"walk BDCOM ONU RX optical power: %w",
			err,
		)
	}

	txRows, err := WalkSubtree(
		client,
		BDCOMONUOpticalTxPowerOID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"walk BDCOM ONU TX optical power: %w",
			err,
		)
	}

	records, err := ParseBDCOMONUOpticalRows(
		rxRows,
		txRows,
	)
	if err != nil {
		return nil, err
	}

	return &ONUOpticalCollection{
		Vendor:    "BDCOM",
		SampledAt: sampledAt,
		Records:   records,
	}, nil
}

func ParseBDCOMONUOpticalRows(
	rxRows []WalkResult,
	txRows []WalkResult,
) ([]ONUOpticalRecord, error) {
	if len(rxRows) == 0 && len(txRows) == 0 {
		return nil, fmt.Errorf(
			"BDCOM ONU optical rows are required",
		)
	}

	records := make(map[int]*ONUOpticalRecord)

	parse := func(
		rows []WalkResult,
		rootOID string,
		assign func(*ONUOpticalRecord, *float64),
	) {
		for _, row := range rows {
			ifIndex, ok := parseBDCOMONUOpticalOID(
				row.OID,
				rootOID,
			)
			if !ok {
				continue
			}

			record := records[ifIndex]
			if record == nil {
				record = &ONUOpticalRecord{
					IfIndex: ifIndex,
				}
				records[ifIndex] = record
			}

			assign(
				record,
				parseBDCOMDeciDBMPointer(
					walkResultText(row.Value),
				),
			)
		}
	}

	parse(
		rxRows,
		BDCOMONUOpticalRxPowerOID,
		func(record *ONUOpticalRecord, value *float64) {
			record.RxPowerDBM = value
		},
	)

	parse(
		txRows,
		BDCOMONUOpticalTxPowerOID,
		func(record *ONUOpticalRecord, value *float64) {
			record.TxPowerDBM = value
		},
	)

	keys := make([]int, 0, len(records))

	for ifIndex, record := range records {
		if record.RxPowerDBM == nil &&
			record.TxPowerDBM == nil {
			continue
		}

		keys = append(keys, ifIndex)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf(
			"BDCOM ONU optical rows contained no usable records",
		)
	}

	sort.Ints(keys)

	result := make(
		[]ONUOpticalRecord,
		0,
		len(keys),
	)

	for _, ifIndex := range keys {
		result = append(
			result,
			*records[ifIndex],
		)
	}

	return result, nil
}

func parseBDCOMONUOpticalOID(
	oid string,
	rootOID string,
) (int, bool) {
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
		return 0, false
	}

	suffix := strings.TrimPrefix(value, prefix)

	if strings.Contains(suffix, ".") {
		return 0, false
	}

	ifIndex, err := strconv.Atoi(suffix)
	if err != nil || ifIndex <= 0 {
		return 0, false
	}

	return ifIndex, true
}

func parseBDCOMDeciDBMPointer(
	value string,
) *float64 {
	raw, err := strconv.ParseInt(
		strings.TrimSpace(value),
		10,
		64,
	)
	if err != nil || raw == -2147483648 {
		return nil
	}

	valueDBM := float64(raw) / 10

	return &valueDBM
}

func MergeBDCOMONUOptical(
	candidates []ONUPersistenceCandidate,
	optical *ONUOpticalCollection,
) []ONUPersistenceCandidate {
	if len(candidates) == 0 ||
		optical == nil ||
		len(optical.Records) == 0 {
		return candidates
	}

	byIfIndex := make(
		map[int]ONUOpticalRecord,
		len(optical.Records),
	)

	for _, record := range optical.Records {
		if record.IfIndex > 0 {
			byIfIndex[record.IfIndex] = record
		}
	}

	for index := range candidates {
		record, ok := byIfIndex[candidates[index].IfIndex]
		if !ok {
			continue
		}

		if record.TxPowerDBM != nil {
			candidates[index].TxPowerDBM =
				record.TxPowerDBM
		}

		if record.RxPowerDBM != nil {
			candidates[index].RxPowerDBM =
				record.RxPowerDBM
		}
	}

	return candidates
}

func (BDCOMONUAdapter) BuildPersistenceCandidates(
	ifmib *IFMIBCollection,
	optical *ONUOpticalCollection,
) ([]ONUPersistenceCandidate, error) {
	if ifmib == nil {
		return nil, fmt.Errorf(
			"IF-MIB collection is required",
		)
	}

	candidates := make(
		[]ONUPersistenceCandidate,
		0,
	)

	for _, port := range ifmib.Ports {
		ponNo, onuNo, ok :=
			parseBDCOMONUIfDescr(port.Name)

		if !ok {
			ponNo, onuNo, ok =
				parseBDCOMONUIfDescr(port.Description)
		}

		if !ok {
			continue
		}

		description := strings.TrimSpace(port.Name)
		if description == "" {
			description = strings.TrimSpace(port.Description)
		}

		candidate := ONUPersistenceCandidate{
			PONNo:       ponNo,
			ONUNo:       onuNo,
			IfIndex:     port.IfIndex,
			Description: description,
			OperStatus:  InterfaceStatus(port.OperStatus),
			SampledAt:   ifmib.SampledAt,
		}

		if port.HCInOctets != nil {
			candidate.InOctets =
				*port.HCInOctets
		}

		if port.HCOutOctets != nil {
			candidate.OutOctets =
				*port.HCOutOctets
		}

		candidates = append(
			candidates,
			candidate,
		)
	}

	sort.Slice(
		candidates,
		func(i, j int) bool {
			if candidates[i].PONNo !=
				candidates[j].PONNo {
				return candidates[i].PONNo <
					candidates[j].PONNo
			}

			return candidates[i].ONUNo <
				candidates[j].ONUNo
		},
	)

	candidates = MergeBDCOMONUOptical(
		candidates,
		optical,
	)

	return candidates, nil
}
