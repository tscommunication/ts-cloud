package snmp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	SZCOMEnterpriseOID = ".1.3.6.1.4.1.12170"

	SZCOMONUNameOID           = ".1.3.6.1.4.1.12170.2.3.4.1.1.2"
	SZCOMONUMACOID            = ".1.3.6.1.4.1.12170.2.3.4.1.1.7"
	SZCOMONUStatusOID         = ".1.3.6.1.4.1.12170.2.3.4.1.1.8"
	SZCOMONUOpticalRxPowerOID = ".1.3.6.1.4.1.12170.2.3.4.2.1.4"
	SZCOMONUOpticalTxPowerOID = ".1.3.6.1.4.1.12170.2.3.4.2.1.5"
)

type SZCOMONUAdapter struct{}

type SZCOMONUInventoryRecord struct {
	Index       uint64
	PONNo       int
	ONUNo       int
	Description string
	MACAddress  string
	OperStatus  string
}

type SZCOMONUInventoryCollection struct {
	Vendor    string
	SampledAt time.Time
	Records   []SZCOMONUInventoryRecord
}

type SZCOMONUInventoryCollector interface {
	CollectInventory(
		cfg V2CConfig,
		sampledAt time.Time,
	) (*SZCOMONUInventoryCollection, error)
}

func (SZCOMONUAdapter) Name() string {
	return "SZCOM"
}

func (SZCOMONUAdapter) Matches(
	vendor string,
	sysObjectID string,
) bool {
	switch normalizedVendor(vendor) {
	case "SZCOM", "PHOTON", "TBS", "TBS PHOTON",
		"SOLITINE", "SOLITINE / TBS":
		return true
	}

	oid := strings.TrimSpace(sysObjectID)

	return oid == SZCOMEnterpriseOID ||
		strings.HasPrefix(oid, SZCOMEnterpriseOID+".")
}

func (SZCOMONUAdapter) CollectInventory(
	cfg V2CConfig,
	sampledAt time.Time,
) (*SZCOMONUInventoryCollection, error) {
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
			index, ok := parseSZCOMInventoryOID(row.OID, rootOID)
			if !ok {
				continue
			}

			assign(get(index), strings.TrimSpace(walkResultText(row.Value)))
		}

		return nil
	}

	if err := collectText(
		SZCOMONUNameOID,
		func(record *partial, value string) {
			record.description = value
		},
	); err != nil {
		return nil, fmt.Errorf("walk SZCOM ONU name: %w", err)
	}

	macRows, err := WalkSubtree(client, SZCOMONUMACOID)
	if err != nil {
		return nil, fmt.Errorf("walk SZCOM ONU MAC: %w", err)
	}

	for _, row := range macRows {
		index, ok := parseSZCOMInventoryOID(row.OID, SZCOMONUMACOID)
		if !ok {
			continue
		}

		get(index).mac = parseECOMMACValue(row.Value)
	}

	if err := collectText(
		SZCOMONUStatusOID,
		func(record *partial, value string) {
			record.status = parseECOMONUStatus(value)
		},
	); err != nil {
		return nil, fmt.Errorf("walk SZCOM ONU status: %w", err)
	}

	out := make([]SZCOMONUInventoryRecord, 0, len(records))

	for _, record := range records {
		ponNo, onuNo, ok := parseSZCOMONUDeviceIndex(record.index)
		if !ok {
			continue
		}

		out = append(out, SZCOMONUInventoryRecord{
			Index:       record.index,
			PONNo:       ponNo,
			ONUNo:       onuNo,
			Description: record.description,
			MACAddress:  record.mac,
			OperStatus:  record.status,
		})
	}

	return &SZCOMONUInventoryCollection{
		Vendor:    "SZCOM",
		SampledAt: sampledAt,
		Records:   out,
	}, nil
}

func parseSZCOMInventoryOID(
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

	if _, _, ok := parseSZCOMONUDeviceIndex(index); !ok {
		return 0, false
	}

	return index, true
}

func parseSZCOMONUDeviceIndex(
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

func BuildSZCOMONUInventoryCandidates(
	inventory *SZCOMONUInventoryCollection,
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

func (SZCOMONUAdapter) CollectOptical(
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

	rxRows, err := WalkSubtree(
		client,
		SZCOMONUOpticalRxPowerOID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"walk SZCOM ONU RX optical power: %w",
			err,
		)
	}

	txRows, err := WalkSubtree(
		client,
		SZCOMONUOpticalTxPowerOID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"walk SZCOM ONU TX optical power: %w",
			err,
		)
	}

	records, err := ParseSZCOMONUOpticalRows(
		rxRows,
		txRows,
	)
	if err != nil {
		return nil, err
	}

	return &ONUOpticalCollection{
		Vendor:    "SZCOM",
		SampledAt: sampledAt,
		Records:   records,
	}, nil
}

func ParseSZCOMONUOpticalRows(
	rxRows []WalkResult,
	txRows []WalkResult,
) ([]ONUOpticalRecord, error) {
	if len(rxRows) == 0 && len(txRows) == 0 {
		return nil, fmt.Errorf(
			"SZCOM ONU optical rows are required",
		)
	}

	records := make(map[int]*ONUOpticalRecord)

	parse := func(
		rows []WalkResult,
		rootOID string,
		assign func(*ONUOpticalRecord, *float64),
	) {
		for _, row := range rows {
			ifIndex, ok := parseSZCOMONUOpticalOID(
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
				parseSZCOMCentiDBMPointer(
					walkResultText(row.Value),
				),
			)
		}
	}

	parse(
		rxRows,
		SZCOMONUOpticalRxPowerOID,
		func(record *ONUOpticalRecord, value *float64) {
			record.RxPowerDBM = value
		},
	)

	parse(
		txRows,
		SZCOMONUOpticalTxPowerOID,
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
			"SZCOM ONU optical rows contained no usable records",
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

func parseSZCOMONUOpticalOID(
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

	deviceIndex, err := strconv.ParseUint(
		suffix,
		10,
		32,
	)
	if err != nil {
		return 0, false
	}

	if _, _, ok := parseSZCOMONUDeviceIndex(
		deviceIndex,
	); !ok {
		return 0, false
	}

	return int(deviceIndex), true
}

func parseSZCOMCentiDBMPointer(
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

	valueDBM := float64(raw) / 100

	return &valueDBM
}

func MergeSZCOMONUOptical(
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

func (SZCOMONUAdapter) BuildPersistenceCandidates(
	ifmib *IFMIBCollection,
	optical *ONUOpticalCollection,
) ([]ONUPersistenceCandidate, error) {
	return nil, nil
}
