package snmp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	HSGQEnterpriseOID = ".1.3.6.1.4.1.50224"

	HSGQONUNameOID   = ".1.3.6.1.4.1.50224.3.3.2.1.2"
	HSGQONUMACOID    = ".1.3.6.1.4.1.50224.3.3.2.1.7"
	HSGQONUStatusOID = ".1.3.6.1.4.1.50224.3.3.2.1.8"
	HSGQONUModelOID  = ".1.3.6.1.4.1.50224.3.3.2.1.26"

	HSGQONUOpticalRxPowerOID = ".1.3.6.1.4.1.50224.3.3.3.1.4"
	HSGQONUOpticalTxPowerOID = ".1.3.6.1.4.1.50224.3.3.3.1.5"
)

type HSGQONUAdapter struct{}

type HSGQONUInventoryRecord struct {
	Index       uint64
	PONNo       int
	ONUNo       int
	Description string
	MACAddress  string
	OperStatus  string
	Model       string
}

type HSGQONUInventoryCollection struct {
	Vendor    string
	SampledAt time.Time
	Records   []HSGQONUInventoryRecord
}

type HSGQONUInventoryCollector interface {
	CollectInventory(
		cfg V2CConfig,
		sampledAt time.Time,
	) (*HSGQONUInventoryCollection, error)
}

func (HSGQONUAdapter) Name() string {
	return "HSGQ"
}

func (HSGQONUAdapter) Matches(
	vendor string,
	sysObjectID string,
) bool {
	if normalizedVendor(vendor) == "HSGQ" {
		return true
	}

	oid := strings.TrimSpace(sysObjectID)

	return oid == HSGQEnterpriseOID ||
		strings.HasPrefix(oid, HSGQEnterpriseOID+".")
}

func (HSGQONUAdapter) CollectInventory(
	cfg V2CConfig,
	sampledAt time.Time,
) (*HSGQONUInventoryCollection, error) {
	if sampledAt.IsZero() {
		return nil, fmt.Errorf("sample time is required")
	}

	client, err := NewV2CClient(cfg)
	if err != nil {
		return nil, err
	}

	type partial struct {
		index       uint64
		description string
		mac         string
		status      string
		model       string
	}

	records := map[uint64]*partial{}

	get := func(index uint64) *partial {
		record := records[index]
		if record == nil {
			record = &partial{index: index}
			records[index] = record
		}
		return record
	}

	collectText := func(
		rootOID string,
		assign func(*partial, string),
	) error {
		rows, err := WalkSubtree(client, rootOID)
		if err != nil {
			return err
		}

		for _, row := range rows {
			index, ok := parseHSGQInventoryOID(row.OID, rootOID)
			if !ok {
				continue
			}

			assign(get(index), strings.TrimSpace(walkResultText(row.Value)))
		}

		return nil
	}

	if err := collectText(
		HSGQONUNameOID,
		func(record *partial, value string) {
			record.description = value
		},
	); err != nil {
		return nil, fmt.Errorf("walk HSGQ ONU name: %w", err)
	}

	macRows, err := WalkSubtree(client, HSGQONUMACOID)
	if err != nil {
		return nil, fmt.Errorf("walk HSGQ ONU MAC: %w", err)
	}

	for _, row := range macRows {
		index, ok := parseHSGQInventoryOID(row.OID, HSGQONUMACOID)
		if !ok {
			continue
		}

		get(index).mac = parseECOMMACValue(row.Value)
	}

	if err := collectText(
		HSGQONUStatusOID,
		func(record *partial, value string) {
			record.status = parseECOMONUStatus(value)
		},
	); err != nil {
		return nil, fmt.Errorf("walk HSGQ ONU status: %w", err)
	}

	if err := collectText(
		HSGQONUModelOID,
		func(record *partial, value string) {
			record.model = value
		},
	); err != nil {
		return nil, fmt.Errorf("walk HSGQ ONU model: %w", err)
	}

	out := make([]HSGQONUInventoryRecord, 0, len(records))

	for _, record := range records {
		ponNo, onuNo, ok := parseHSGQONUDeviceIndex(record.index)
		if !ok {
			continue
		}

		out = append(out, HSGQONUInventoryRecord{
			Index:       record.index,
			PONNo:       ponNo,
			ONUNo:       onuNo,
			Description: record.description,
			MACAddress:  record.mac,
			OperStatus:  record.status,
			Model:       record.model,
		})
	}

	return &HSGQONUInventoryCollection{
		Vendor:    "HSGQ",
		SampledAt: sampledAt,
		Records:   out,
	}, nil
}

func parseHSGQInventoryOID(
	oid string,
	rootOID string,
) (uint64, bool) {
	root := strings.TrimPrefix(strings.TrimSpace(rootOID), ".")
	value := strings.TrimPrefix(strings.TrimSpace(oid), ".")
	prefix := root + "."

	if !strings.HasPrefix(value, prefix) {
		return 0, false
	}

	suffix := strings.TrimPrefix(value, prefix)

	if strings.Contains(suffix, ".") {
		return 0, false
	}

	index, err := strconv.ParseUint(suffix, 10, 32)
	if err != nil {
		return 0, false
	}

	if _, _, ok := parseHSGQONUDeviceIndex(index); !ok {
		return 0, false
	}

	return index, true
}

func BuildHSGQONUInventoryCandidates(
	inventory *HSGQONUInventoryCollection,
) []ONUPersistenceCandidate {
	if inventory == nil || len(inventory.Records) == 0 {
		return nil
	}

	candidates := make(
		[]ONUPersistenceCandidate,
		0,
		len(inventory.Records),
	)

	for _, record := range inventory.Records {
		if record.PONNo <= 0 || record.ONUNo <= 0 {
			continue
		}

		candidates = append(candidates, ONUPersistenceCandidate{
			PONNo:       record.PONNo,
			ONUNo:       record.ONUNo,
			IfIndex:     int(record.Index),
			Description: strings.TrimSpace(record.Description),
			OperStatus:  record.OperStatus,
			MACAddress:  strings.TrimSpace(record.MACAddress),
			Model:       strings.TrimSpace(record.Model),
			SampledAt:   inventory.SampledAt,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].PONNo != candidates[j].PONNo {
			return candidates[i].PONNo < candidates[j].PONNo
		}

		return candidates[i].ONUNo < candidates[j].ONUNo
	})

	return candidates
}

func (HSGQONUAdapter) CollectOptical(
	cfg V2CConfig,
	sampledAt time.Time,
) (*ONUOpticalCollection, error) {
	if sampledAt.IsZero() {
		return nil, fmt.Errorf("sample time is required")
	}

	collect := func(rootOID string) ([]WalkResult, error) {
		client, err := NewV2CClient(cfg)
		if err != nil {
			return nil, err
		}

		return WalkSubtree(client, rootOID)
	}

	rxRows, err := collect(HSGQONUOpticalRxPowerOID)
	if err != nil {
		return nil, fmt.Errorf("walk HSGQ ONU RX optical power: %w", err)
	}

	txRows, err := collect(HSGQONUOpticalTxPowerOID)
	if err != nil {
		return nil, fmt.Errorf("walk HSGQ ONU TX optical power: %w", err)
	}

	records, err := ParseHSGQONUOpticalRows(rxRows, txRows)
	if err != nil {
		return nil, err
	}

	return &ONUOpticalCollection{
		Vendor:    "HSGQ",
		SampledAt: sampledAt,
		Records:   records,
	}, nil
}

func ParseHSGQONUOpticalRows(
	rxRows []WalkResult,
	txRows []WalkResult,
) ([]ONUOpticalRecord, error) {
	if len(rxRows) == 0 && len(txRows) == 0 {
		return nil, fmt.Errorf("HSGQ ONU optical rows are required")
	}

	records := make(map[int]*ONUOpticalRecord)

	parse := func(
		rows []WalkResult,
		rootOID string,
		assign func(*ONUOpticalRecord, *float64),
	) {
		for _, row := range rows {
			deviceIndex, ok := parseHSGQONUOpticalOID(
				row.OID,
				rootOID,
			)
			if !ok {
				continue
			}

			record := records[deviceIndex]
			if record == nil {
				record = &ONUOpticalRecord{
					IfIndex: deviceIndex,
				}
				records[deviceIndex] = record
			}

			assign(
				record,
				parseHSGQCentiDBMPointer(
					walkResultText(row.Value),
				),
			)
		}
	}

	parse(
		rxRows,
		HSGQONUOpticalRxPowerOID,
		func(record *ONUOpticalRecord, value *float64) {
			record.RxPowerDBM = value
		},
	)

	parse(
		txRows,
		HSGQONUOpticalTxPowerOID,
		func(record *ONUOpticalRecord, value *float64) {
			record.TxPowerDBM = value
		},
	)

	keys := make([]int, 0, len(records))
	for deviceIndex, record := range records {
		if record.RxPowerDBM == nil &&
			record.TxPowerDBM == nil {
			continue
		}

		keys = append(keys, deviceIndex)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf(
			"HSGQ ONU optical rows contained no usable records",
		)
	}

	sort.Ints(keys)

	result := make([]ONUOpticalRecord, 0, len(keys))
	for _, deviceIndex := range keys {
		result = append(result, *records[deviceIndex])
	}

	return result, nil
}

func parseHSGQONUOpticalOID(
	oid string,
	rootOID string,
) (int, bool) {
	root := strings.TrimPrefix(strings.TrimSpace(rootOID), ".")
	value := strings.TrimPrefix(strings.TrimSpace(oid), ".")
	prefix := root + "."

	if !strings.HasPrefix(value, prefix) {
		return 0, false
	}

	suffix := strings.Split(strings.TrimPrefix(value, prefix), ".")
	if len(suffix) != 3 || suffix[1] != "0" || suffix[2] != "0" {
		return 0, false
	}

	deviceIndex, err := strconv.ParseUint(suffix[0], 10, 32)
	if err != nil {
		return 0, false
	}

	ponNo, onuNo, ok := parseHSGQONUDeviceIndex(deviceIndex)
	if !ok || ponNo <= 0 || onuNo <= 0 {
		return 0, false
	}

	return int(deviceIndex), true
}

func parseHSGQONUDeviceIndex(
	deviceIndex uint64,
) (
	ponNo int,
	onuNo int,
	ok bool,
) {
	if deviceIndex == 0 || deviceIndex > 0xffffffff {
		return 0, 0, false
	}

	ponNo = int((deviceIndex >> 8) & 0xff)
	onuNo = int(deviceIndex & 0xff)

	if ponNo <= 0 || onuNo <= 0 {
		return 0, 0, false
	}

	return ponNo, onuNo, true
}

func parseHSGQCentiDBMPointer(value string) *float64 {
	raw, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || raw == -2147483648 {
		return nil
	}

	valueDBM := float64(raw) / 100

	return &valueDBM
}

func MergeHSGQONUOptical(
	candidates []ONUPersistenceCandidate,
	optical *ONUOpticalCollection,
) []ONUPersistenceCandidate {
	if len(candidates) == 0 ||
		optical == nil ||
		len(optical.Records) == 0 {
		return candidates
	}

	byIfIndex := make(map[int]ONUOpticalRecord, len(optical.Records))
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
			candidates[index].TxPowerDBM = record.TxPowerDBM
		}

		if record.RxPowerDBM != nil {
			candidates[index].RxPowerDBM = record.RxPowerDBM
		}
	}

	return candidates
}

func (HSGQONUAdapter) BuildPersistenceCandidates(
	ifmib *IFMIBCollection,
	optical *ONUOpticalCollection,
) ([]ONUPersistenceCandidate, error) {
	return nil, nil
}
