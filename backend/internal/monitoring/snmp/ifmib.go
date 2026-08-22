package snmp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	IFDescrOID       = ".1.3.6.1.2.1.2.2.1.2"
	IFTypeOID        = ".1.3.6.1.2.1.2.2.1.3"
	IFSpeedOID       = ".1.3.6.1.2.1.2.2.1.5"
	IFPhysAddressOID = ".1.3.6.1.2.1.2.2.1.6"
	IFAdminStatusOID = ".1.3.6.1.2.1.2.2.1.7"
	IFOperStatusOID  = ".1.3.6.1.2.1.2.2.1.8"
	IFLastChangeOID  = ".1.3.6.1.2.1.2.2.1.9"

	IFInDiscardsOID  = ".1.3.6.1.2.1.2.2.1.13"
	IFInErrorsOID    = ".1.3.6.1.2.1.2.2.1.14"
	IFOutDiscardsOID = ".1.3.6.1.2.1.2.2.1.19"
	IFOutErrorsOID   = ".1.3.6.1.2.1.2.2.1.20"

	IFNameOID        = ".1.3.6.1.2.1.31.1.1.1.1"
	IFHCInOctetsOID  = ".1.3.6.1.2.1.31.1.1.1.6"
	IFHCOutOctetsOID = ".1.3.6.1.2.1.31.1.1.1.10"
	IFHighSpeedOID   = ".1.3.6.1.2.1.31.1.1.1.15"

	SysUpTimeOID = ".1.3.6.1.2.1.1.3.0"
)

type IFMIBPort struct {
	IfIndex int

	Name        string
	Description string
	IfType      int

	AdminStatus int
	OperStatus  int

	HighSpeedMbps uint64
	SpeedBps      uint64

	MACAddress string

	LastChangeTicks uint64

	HCInOctets  *uint64
	HCOutOctets *uint64

	InErrors    *uint64
	OutErrors   *uint64
	InDiscards  *uint64
	OutDiscards *uint64
}

func IFIndexFromOID(rootOID, oid string) (int, error) {
	root := strings.TrimSuffix(strings.TrimSpace(rootOID), ".")
	value := strings.TrimSpace(oid)

	prefix := root + "."
	if !strings.HasPrefix(value, prefix) {
		return 0, fmt.Errorf(
			"OID %q is outside subtree %q",
			value,
			root,
		)
	}

	suffix := strings.TrimPrefix(value, prefix)
	if suffix == "" || strings.Contains(suffix, ".") {
		return 0, fmt.Errorf("invalid interface index suffix %q", suffix)
	}

	index, err := strconv.Atoi(suffix)
	if err != nil || index <= 0 {
		return 0, fmt.Errorf("invalid interface index %q", suffix)
	}

	return index, nil
}

func StringValue(value interface{}) (string, error) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), nil
	case []byte:
		return strings.TrimSpace(string(typed)), nil
	default:
		return "", fmt.Errorf(
			"unsupported string value type %T",
			value,
		)
	}
}

func Uint64Value(value interface{}) (uint64, error) {
	switch typed := value.(type) {
	case uint64:
		return typed, nil
	case uint32:
		return uint64(typed), nil
	case uint:
		return uint64(typed), nil
	case int:
		if typed < 0 {
			return 0, errors.New("negative integer")
		}
		return uint64(typed), nil
	case int32:
		if typed < 0 {
			return 0, errors.New("negative integer")
		}
		return uint64(typed), nil
	case int64:
		if typed < 0 {
			return 0, errors.New("negative integer")
		}
		return uint64(typed), nil
	default:
		return 0, fmt.Errorf(
			"unsupported numeric value type %T",
			value,
		)
	}
}

func IntValue(value interface{}) (int, error) {
	numeric, err := Uint64Value(value)
	if err != nil {
		return 0, err
	}

	maxInt := uint64(^uint(0) >> 1)
	if numeric > maxInt {
		return 0, errors.New("integer overflow")
	}

	return int(numeric), nil
}

func MACAddressValue(value interface{}) (string, error) {
	raw, ok := value.([]byte)
	if !ok {
		return "", fmt.Errorf(
			"unsupported MAC value type %T",
			value,
		)
	}

	if len(raw) == 0 {
		return "", nil
	}

	encoded := strings.ToUpper(hex.EncodeToString(raw))

	if len(raw) != 6 {
		return encoded, nil
	}

	return strings.Join([]string{
		encoded[0:2],
		encoded[2:4],
		encoded[4:6],
		encoded[6:8],
		encoded[8:10],
		encoded[10:12],
	}, ":"), nil
}

func EffectiveSpeedMbps(
	highSpeedMbps uint64,
	speedBps uint64,
) int64 {
	if highSpeedMbps > 0 {
		if highSpeedMbps > uint64(^uint64(0)>>1) {
			return 0
		}
		return int64(highSpeedMbps)
	}

	if speedBps == 0 {
		return 0
	}

	return int64(speedBps / 1_000_000)
}

type IFMIBColumn struct {
	RootOID string
	Rows    []WalkResult
}

func MergeIFMIBColumns(
	columns []IFMIBColumn,
) (map[int]*IFMIBPort, error) {
	ports := make(map[int]*IFMIBPort)

	portFor := func(index int) *IFMIBPort {
		if existing, ok := ports[index]; ok {
			return existing
		}

		port := &IFMIBPort{
			IfIndex: index,
		}

		ports[index] = port
		return port
	}

	for _, column := range columns {
		for _, row := range column.Rows {
			index, err := IFIndexFromOID(
				column.RootOID,
				row.OID,
			)
			if err != nil {
				return nil, err
			}

			port := portFor(index)

			switch column.RootOID {
			case IFNameOID:
				value, err := StringValue(row.Value)
				if err != nil {
					return nil, err
				}
				port.Name = value

			case IFDescrOID:
				value, err := StringValue(row.Value)
				if err != nil {
					return nil, err
				}
				port.Description = value

			case IFTypeOID:
				value, err := IntValue(row.Value)
				if err != nil {
					return nil, err
				}
				port.IfType = value

			case IFAdminStatusOID:
				value, err := IntValue(row.Value)
				if err != nil {
					return nil, err
				}
				port.AdminStatus = value

			case IFOperStatusOID:
				value, err := IntValue(row.Value)
				if err != nil {
					return nil, err
				}
				port.OperStatus = value

			case IFHighSpeedOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.HighSpeedMbps = value

			case IFSpeedOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.SpeedBps = value

			case IFPhysAddressOID:
				value, err := MACAddressValue(row.Value)
				if err != nil {
					return nil, err
				}
				port.MACAddress = value

			case IFLastChangeOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.LastChangeTicks = value

			case IFHCInOctetsOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.HCInOctets = uint64Pointer(value)

			case IFHCOutOctetsOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.HCOutOctets = uint64Pointer(value)

			case IFInErrorsOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.InErrors = uint64Pointer(value)

			case IFOutErrorsOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.OutErrors = uint64Pointer(value)

			case IFInDiscardsOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.InDiscards = uint64Pointer(value)

			case IFOutDiscardsOID:
				value, err := Uint64Value(row.Value)
				if err != nil {
					return nil, err
				}
				port.OutDiscards = uint64Pointer(value)
			}
		}
	}

	return ports, nil
}

func uint64Pointer(value uint64) *uint64 {
	return &value
}

func InterfaceStatus(value int) string {
	switch value {
	case 1:
		return "UP"
	case 2:
		return "DOWN"
	case 3:
		return "TESTING"
	case 4:
		return "UNKNOWN"
	case 5:
		return "DORMANT"
	case 6:
		return "NOT_PRESENT"
	case 7:
		return "LOWER_LAYER_DOWN"
	default:
		return "UNKNOWN"
	}
}

func LastChangeAt(
	sampledAt time.Time,
	sysUpTimeTicks uint64,
	lastChangeTicks uint64,
) *time.Time {
	if lastChangeTicks == 0 {
		return nil
	}

	if sysUpTimeTicks == 0 {
		return nil
	}

	if lastChangeTicks > sysUpTimeTicks {
		return nil
	}

	elapsedTicks := sysUpTimeTicks - lastChangeTicks

	elapsed := time.Duration(elapsedTicks) * 10 * time.Millisecond

	changedAt := sampledAt.Add(-elapsed)

	return &changedAt
}
