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

const (
	ECOMEnterpriseOID = ".1.3.6.1.4.1.17409"

	ECOMONUNameOID           = ".1.3.6.1.4.1.17409.2.3.4.1.1.2"
	ECOMONUMACOID            = ".1.3.6.1.4.1.17409.2.3.4.1.1.7"
	ECOMONUStatusOID         = ".1.3.6.1.4.1.17409.2.3.4.1.1.8"
	ECOMONUOpticalRxPowerOID = ".1.3.6.1.4.1.17409.2.3.4.2.1.4"
	ECOMONUOpticalTxPowerOID = ".1.3.6.1.4.1.17409.2.3.4.2.1.5"
	ECOMSniMacAddressTypeOID = ".1.3.6.1.4.1.17409.2.3.2.4.2.1.3"
	ECOMSniMacAddressPortOID = ".1.3.6.1.4.1.17409.2.3.2.4.2.1.4"
)

type ECOMONUAdapter struct{}

func (ECOMONUAdapter) Name() string {
	return "ECOM"
}

func (ECOMONUAdapter) Matches(
	vendor string,
	sysObjectID string,
) bool {
	if normalizedVendor(vendor) == "ECOM" ||
		normalizedVendor(vendor) == "C-DATA" ||
		normalizedVendor(vendor) == "CDATA" {
		return true
	}

	oid := strings.TrimSpace(sysObjectID)

	return oid == ECOMEnterpriseOID ||
		strings.HasPrefix(
			oid,
			ECOMEnterpriseOID+".",
		)
}

type ECOMONUInventoryRecord struct {
	Index       uint64
	PONNo       int
	ONUNo       int
	Description string
	MACAddress  string
	OperStatus  string
}

type ECOMONUInventoryCollection struct {
	Vendor    string
	SampledAt time.Time
	Records   []ECOMONUInventoryRecord
}

type ECOMONUInventoryCollector interface {
	CollectInventory(
		cfg V2CConfig,
		sampledAt time.Time,
	) (*ECOMONUInventoryCollection, error)
}

func (ECOMONUAdapter) CollectInventory(
	cfg V2CConfig,
	sampledAt time.Time,
) (*ECOMONUInventoryCollection, error) {
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
		r := records[index]
		if r == nil {
			r = &partial{index: index}
			records[index] = r
		}
		return r
	}

	nameRows, err := WalkSubtree(client, ECOMONUNameOID)
	if err != nil {
		return nil, fmt.Errorf("walk ECOM ONU name: %w", err)
	}

	for _, row := range nameRows {
		index, ok := parseECOMIndexOID(row.OID, ECOMONUNameOID)
		if !ok {
			continue
		}

		var value string
		switch v := row.Value.(type) {
		case []byte:
			value = strings.TrimSpace(string(v))
		default:
			value = strings.TrimSpace(fmt.Sprint(v))
		}

		get(index).description = value
	}

	macRows, err := WalkSubtree(client, ECOMONUMACOID)
	if err != nil {
		return nil, fmt.Errorf("walk ECOM ONU MAC: %w", err)
	}

	for _, row := range macRows {
		index, ok := parseECOMIndexOID(row.OID, ECOMONUMACOID)
		if !ok {
			continue
		}

		get(index).mac = parseECOMMACValue(row.Value)
	}

	statusRows, err := WalkSubtree(client, ECOMONUStatusOID)
	if err != nil {
		return nil, fmt.Errorf("walk ECOM ONU status: %w", err)
	}

	for _, row := range statusRows {
		index, ok := parseECOMIndexOID(row.OID, ECOMONUStatusOID)
		if !ok {
			continue
		}

		get(index).status = parseECOMONUStatus(
			fmt.Sprint(row.Value),
		)
	}

	out := make([]ECOMONUInventoryRecord, 0, len(records))

	for _, r := range records {
		ponNo, onuNo, ok := parseECOMONUDescription(
			r.description,
		)
		if !ok {
			continue
		}

		out = append(out, ECOMONUInventoryRecord{
			Index:       r.index,
			PONNo:       ponNo,
			ONUNo:       onuNo,
			Description: r.description,
			MACAddress:  r.mac,
			OperStatus:  r.status,
		})
	}

	return &ECOMONUInventoryCollection{
		Vendor:    "ECOM",
		SampledAt: sampledAt,
		Records:   out,
	}, nil
}

var ecomONUDescriptionRE = regexp.MustCompile(
	`(?i)^epon\s+\d+/\d+/(\d+)\s+onu\s+(\d+)(?:\s+.*)?$`,
)

func parseECOMONUDescription(
	value string,
) (
	ponNo int,
	onuNo int,
	ok bool,
) {
	match := ecomONUDescriptionRE.FindStringSubmatch(
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

func parseECOMONUStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "1":
		return "UP"
	case "2":
		return "DOWN"
	default:
		return "UNKNOWN"
	}
}

func parseECOMMACValue(value any) string {
	raw, ok := value.([]byte)

	if !ok || len(raw) != 6 {
		return ""
	}

	return fmt.Sprintf(
		"%02X:%02X:%02X:%02X:%02X:%02X",
		raw[0],
		raw[1],
		raw[2],
		raw[3],
		raw[4],
		raw[5],
	)
}

func parseECOMIndexOID(
	oid string,
	rootOID string,
) (
	uint64,
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

	index, err := strconv.ParseUint(
		suffix,
		10,
		64,
	)

	if err != nil || index == 0 {
		return 0, false
	}

	return index, true
}

func BuildECOMONUInventoryCandidates(
	inventory *ECOMONUInventoryCollection,
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

		candidates = append(
			candidates,
			ONUPersistenceCandidate{
				PONNo:       record.PONNo,
				ONUNo:       record.ONUNo,
				IfIndex:     int(record.Index),
				Description: strings.TrimSpace(record.Description),
				OperStatus:  record.OperStatus,
				MACAddress:  strings.TrimSpace(record.MACAddress),
				SampledAt:   inventory.SampledAt,
			},
		)
	}

	sort.Slice(
		candidates,
		func(i, j int) bool {
			if candidates[i].PONNo != candidates[j].PONNo {
				return candidates[i].PONNo < candidates[j].PONNo
			}

			return candidates[i].ONUNo < candidates[j].ONUNo
		},
	)

	return candidates
}

func (ECOMONUAdapter) CollectOptical(
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

	rxRows, err := collect(ECOMONUOpticalRxPowerOID)
	if err != nil {
		return nil, fmt.Errorf("walk ECOM ONU RX optical power: %w", err)
	}

	txRows, err := collect(ECOMONUOpticalTxPowerOID)
	if err != nil {
		return nil, fmt.Errorf("walk ECOM ONU TX optical power: %w", err)
	}

	records, err := ParseECOMONUOpticalRows(rxRows, txRows)
	if err != nil {
		return nil, err
	}

	return &ONUOpticalCollection{
		Vendor:    "ECOM",
		SampledAt: sampledAt,
		Records:   records,
	}, nil
}

func ParseECOMONUOpticalRows(
	rxRows []WalkResult,
	txRows []WalkResult,
) ([]ONUOpticalRecord, error) {
	if len(rxRows) == 0 && len(txRows) == 0 {
		return nil, fmt.Errorf("ECOM ONU optical rows are required")
	}

	records := make(map[int]*ONUOpticalRecord)

	parse := func(
		rows []WalkResult,
		rootOID string,
		assign func(*ONUOpticalRecord, *float64),
	) {
		for _, row := range rows {
			ifIndex, ok := parseECOMONUOpticalOID(row.OID, rootOID)
			if !ok {
				continue
			}

			record := records[ifIndex]
			if record == nil {
				record = &ONUOpticalRecord{IfIndex: ifIndex}
				records[ifIndex] = record
			}

			assign(record, parseECOMCentiDBMPointer(walkResultText(row.Value)))
		}
	}

	parse(rxRows, ECOMONUOpticalRxPowerOID, func(
		record *ONUOpticalRecord,
		value *float64,
	) {
		record.RxPowerDBM = value
	})

	parse(txRows, ECOMONUOpticalTxPowerOID, func(
		record *ONUOpticalRecord,
		value *float64,
	) {
		record.TxPowerDBM = value
	})

	if len(records) == 0 {
		return nil, fmt.Errorf("ECOM ONU optical rows contained no usable records")
	}

	keys := make([]int, 0, len(records))
	for ifIndex := range records {
		keys = append(keys, ifIndex)
	}
	sort.Ints(keys)

	result := make([]ONUOpticalRecord, 0, len(keys))
	for _, ifIndex := range keys {
		result = append(result, *records[ifIndex])
	}

	return result, nil
}

func parseECOMONUOpticalOID(
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
	if len(suffix) != 3 || suffix[1] != "1" || suffix[2] != "1" {
		return 0, false
	}

	ifIndex, err := strconv.Atoi(suffix[0])
	if err != nil || ifIndex <= 0 {
		return 0, false
	}

	return ifIndex, true
}

func parseECOMCentiDBMPointer(value string) *float64 {
	raw, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return nil
	}

	valueDBM := raw / 100
	return &valueDBM
}

func MergeECOMONUOptical(
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

		if record.TxPowerDBM != nil {
			candidates[index].TxPowerDBM = record.TxPowerDBM
		}

		if record.RxPowerDBM != nil {
			candidates[index].RxPowerDBM = record.RxPowerDBM
		}
	}

	return candidates
}

func (ECOMONUAdapter) BuildPersistenceCandidates(
	ifmib *IFMIBCollection,
	optical *ONUOpticalCollection,
) ([]ONUPersistenceCandidate, error) {
	return nil, nil
}

type ECOMLearnedMACRecord struct {
	MACAddress string
	VLAN       int
	MACType    int
	PortID     int
}

func parseECOMSniMACIndex(
	oid string,
	rootOID string,
) (string, int, bool) {
	prefix := strings.TrimSuffix(rootOID, ".") + "."
	if !strings.HasPrefix(oid, prefix) {
		return "", 0, false
	}

	suffix := strings.TrimPrefix(oid, prefix)
	parts := strings.Split(suffix, ".")

	// SNI MAC table index:
	// <mac-length=6>.<octet1>...<octet6>.<vlan>
	if len(parts) != 8 || parts[0] != "6" {
		return "", 0, false
	}

	octets := make([]byte, 6)
	for i := 0; i < 6; i++ {
		value, err := strconv.Atoi(parts[i+1])
		if err != nil || value < 0 || value > 255 {
			return "", 0, false
		}
		octets[i] = byte(value)
	}

	vlan, err := strconv.Atoi(parts[7])
	if err != nil || vlan < 0 || vlan > 4094 {
		return "", 0, false
	}

	mac := strings.ToUpper(net.HardwareAddr(octets).String())
	return mac, vlan, true
}

func ParseECOMSniMACRecord(
	typeOID string,
	typeValue int,
	portOID string,
	portValue int,
) (*ECOMLearnedMACRecord, error) {
	typeMAC, typeVLAN, ok := parseECOMSniMACIndex(
		typeOID,
		ECOMSniMacAddressTypeOID,
	)
	if !ok {
		return nil, fmt.Errorf("invalid ECOM SNI MAC type OID")
	}

	portMAC, portVLAN, ok := parseECOMSniMACIndex(
		portOID,
		ECOMSniMacAddressPortOID,
	)
	if !ok {
		return nil, fmt.Errorf("invalid ECOM SNI MAC port OID")
	}

	if typeMAC != portMAC || typeVLAN != portVLAN {
		return nil, fmt.Errorf("ECOM SNI MAC indexes do not match")
	}

	if portValue <= 0 {
		return nil, fmt.Errorf("ECOM SNI MAC port ID is required")
	}

	return &ECOMLearnedMACRecord{
		MACAddress: typeMAC,
		VLAN:       typeVLAN,
		MACType:    typeValue,
		PortID:     portValue,
	}, nil
}

func ParseECOMSniMACRows(
	typeRows []WalkResult,
	portRows []WalkResult,
) ([]ECOMLearnedMACRecord, error) {
	typeByKey := make(map[string]int)
	portByKey := make(map[string]int)

	keyFor := func(mac string, vlan int) string {
		return fmt.Sprintf("%s|%d", mac, vlan)
	}

	for _, row := range typeRows {
		mac, vlan, ok := parseECOMSniMACIndex(
			row.OID,
			ECOMSniMacAddressTypeOID,
		)
		if !ok {
			continue
		}

		value, err := strconv.Atoi(
			strings.TrimSpace(walkResultText(row.Value)),
		)
		if err != nil {
			continue
		}

		typeByKey[keyFor(mac, vlan)] = value
	}

	for _, row := range portRows {
		mac, vlan, ok := parseECOMSniMACIndex(
			row.OID,
			ECOMSniMacAddressPortOID,
		)
		if !ok {
			continue
		}

		value, err := strconv.Atoi(
			strings.TrimSpace(walkResultText(row.Value)),
		)
		if err != nil || value <= 0 {
			continue
		}

		portByKey[keyFor(mac, vlan)] = value
	}

	records := make([]ECOMLearnedMACRecord, 0)

	for key, portID := range portByKey {
		macType, ok := typeByKey[key]
		if !ok {
			continue
		}

		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}

		vlan, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		records = append(records, ECOMLearnedMACRecord{
			MACAddress: parts[0],
			VLAN:       vlan,
			MACType:    macType,
			PortID:     portID,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].MACAddress != records[j].MACAddress {
			return records[i].MACAddress < records[j].MACAddress
		}
		if records[i].VLAN != records[j].VLAN {
			return records[i].VLAN < records[j].VLAN
		}
		return records[i].PortID < records[j].PortID
	})

	return records, nil
}

func FindECOMLearnedMAC(
	records []ECOMLearnedMACRecord,
	macAddress string,
) (*ECOMLearnedMACRecord, bool) {
	parsed, err := net.ParseMAC(strings.TrimSpace(macAddress))
	if err != nil {
		return nil, false
	}

	target := strings.ToUpper(parsed.String())

	for i := range records {
		if strings.EqualFold(records[i].MACAddress, target) {
			record := records[i]
			return &record, true
		}
	}

	return nil, false
}

var ecomPONInterfaceRE = regexp.MustCompile(
	`(?i)^epon\s+\d+/\d+/(\d+)$`,
)

func ParseECOMPONInterface(value string) (int, bool) {
	match := ecomPONInterfaceRE.FindStringSubmatch(
		strings.TrimSpace(value),
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

type ECOMLearnedMACResolution struct {
	MACAddress string
	VLAN       int
	MACType    int
	PortID     int
	Interface  string
	PONNo      int
}

type ecomSNIWalkFunc func(rootOID string) ([]WalkResult, error)
type ecomSNIGetFunc func(oid string) (*ProbeResult, error)

func (ECOMONUAdapter) ResolveLearnedMAC(
	cfg V2CConfig,
	macAddress string,
) (*ECOMLearnedMACResolution, error) {
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

	return resolveECOMLearnedMAC(macAddress, walk, get)
}

func resolveECOMLearnedMAC(
	macAddress string,
	walk ecomSNIWalkFunc,
	get ecomSNIGetFunc,
) (*ECOMLearnedMACResolution, error) {
	if _, err := net.ParseMAC(strings.TrimSpace(macAddress)); err != nil {
		return nil, fmt.Errorf("invalid customer MAC address: %w", err)
	}
	if walk == nil {
		return nil, fmt.Errorf("ECOM SNI walk function is required")
	}
	if get == nil {
		return nil, fmt.Errorf("ECOM SNI get function is required")
	}

	typeRows, err := walk(ECOMSniMacAddressTypeOID)
	if err != nil {
		return nil, fmt.Errorf("walk ECOM SNI MAC type: %w", err)
	}

	portRows, err := walk(ECOMSniMacAddressPortOID)
	if err != nil {
		return nil, fmt.Errorf("walk ECOM SNI MAC port: %w", err)
	}

	records, err := ParseECOMSniMACRows(typeRows, portRows)
	if err != nil {
		return nil, err
	}

	record, ok := FindECOMLearnedMAC(records, macAddress)
	if !ok {
		return nil, nil
	}

	result, err := get(
		fmt.Sprintf("%s.%d", IFNameOID, record.PortID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get ECOM learned MAC interface name: %w",
			err,
		)
	}
	if result == nil {
		return nil, fmt.Errorf(
			"ECOM learned MAC interface response is nil",
		)
	}

	interfaceName, err := StringValue(result.Value)
	if err != nil {
		return nil, fmt.Errorf(
			"parse ECOM learned MAC interface name: %w",
			err,
		)
	}

	ponNo, ok := ParseECOMPONInterface(interfaceName)
	if !ok {
		return nil, fmt.Errorf(
			"ECOM learned MAC port %d resolved to unsupported interface %q",
			record.PortID,
			interfaceName,
		)
	}

	return &ECOMLearnedMACResolution{
		MACAddress: record.MACAddress,
		VLAN:       record.VLAN,
		MACType:    record.MACType,
		PortID:     record.PortID,
		Interface:  interfaceName,
		PONNo:      ponNo,
	}, nil
}
