package snmp

import (
	"fmt"
	"time"
)

type PortPersistenceCandidate struct {
	PortKey      string
	IfIndex      int
	Name         string
	Description  string
	PortType     string
	AdminStatus  string
	OperStatus   string
	SpeedMbps    int64
	MACAddress   string
	LastChangeAt *time.Time
	InOctets     uint64
	OutOctets    uint64
	InErrors     uint64
	OutErrors    uint64
	InDiscards   uint64
	OutDiscards  uint64
	SampledAt    time.Time
}

func BuildPortPersistenceCandidate(
	collection *IFMIBCollection,
	port IFMIBPort,
) (PortPersistenceCandidate, error) {
	if collection == nil {
		return PortPersistenceCandidate{},
			fmt.Errorf("IF-MIB collection is required")
	}

	if port.IfIndex <= 0 {
		return PortPersistenceCandidate{},
			fmt.Errorf("invalid ifIndex %d", port.IfIndex)
	}

	if port.HCInOctets == nil {
		return PortPersistenceCandidate{},
			fmt.Errorf(
				"ifIndex %d missing HC in octets",
				port.IfIndex,
			)
	}

	if port.HCOutOctets == nil {
		return PortPersistenceCandidate{},
			fmt.Errorf(
				"ifIndex %d missing HC out octets",
				port.IfIndex,
			)
	}

	return PortPersistenceCandidate{
		PortKey: fmt.Sprintf(
			"ifindex:%d",
			port.IfIndex,
		),
		IfIndex:     port.IfIndex,
		Name:        port.Name,
		Description: port.Description,
		PortType: InterfacePortType(
			port.IfType,
			port.Name,
		),
		AdminStatus: InterfaceStatus(
			port.AdminStatus,
		),
		OperStatus: InterfaceStatus(
			port.OperStatus,
		),
		SpeedMbps: EffectiveSpeedMbps(
			port.HighSpeedMbps,
			port.SpeedBps,
		),
		MACAddress: port.MACAddress,
		LastChangeAt: LastChangeAt(
			collection.SampledAt,
			collection.SysUpTimeTicks,
			port.LastChangeTicks,
		),
		InOctets:    CounterValue(port.HCInOctets),
		OutOctets:   CounterValue(port.HCOutOctets),
		InErrors:    CounterValue(port.InErrors),
		OutErrors:   CounterValue(port.OutErrors),
		InDiscards:  CounterValue(port.InDiscards),
		OutDiscards: CounterValue(port.OutDiscards),
		SampledAt:   collection.SampledAt,
	}, nil
}

func BuildPortPersistenceCandidates(
	collection *IFMIBCollection,
) ([]PortPersistenceCandidate, error) {
	if collection == nil {
		return nil, fmt.Errorf("IF-MIB collection is required")
	}

	candidates := make(
		[]PortPersistenceCandidate,
		0,
		len(collection.Ports),
	)

	for _, port := range collection.Ports {
		candidate, err := BuildPortPersistenceCandidate(
			collection,
			port,
		)
		if err != nil {
			return nil, err
		}

		candidates = append(
			candidates,
			candidate,
		)
	}

	return candidates, nil
}

type PortSampleRateCandidate struct {
	InMbps  float64
	OutMbps float64
}

func BuildPortSampleRateCandidate(
	previousSampledAt time.Time,
	previousInOctets uint64,
	previousOutOctets uint64,
	current PortPersistenceCandidate,
) PortSampleRateCandidate {
	elapsed := current.SampledAt.Sub(previousSampledAt)

	return PortSampleRateCandidate{
		InMbps: MbpsFromOctetDelta(
			previousInOctets,
			current.InOctets,
			elapsed,
		),
		OutMbps: MbpsFromOctetDelta(
			previousOutOctets,
			current.OutOctets,
			elapsed,
		),
	}
}
