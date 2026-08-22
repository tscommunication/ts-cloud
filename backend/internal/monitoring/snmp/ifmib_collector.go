package snmp

import (
	"fmt"
	"sort"
	"time"
)

type IFMIBCollection struct {
	SampledAt      time.Time
	SysUpTimeTicks uint64
	Ports          []IFMIBPort
	Warnings       []string
}

type ifMIBWalkFunc func(rootOID string) ([]WalkResult, error)
type ifMIBGetFunc func(oid string) (*ProbeResult, error)

type ifMIBColumnSpec struct {
	RootOID  string
	Required bool
}

func CollectIFMIBV2C(
	cfg V2CConfig,
	sampledAt time.Time,
) (*IFMIBCollection, error) {
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

	return collectIFMIB(
		sampledAt,
		walk,
		get,
	)
}

func collectIFMIB(
	sampledAt time.Time,
	walk ifMIBWalkFunc,
	get ifMIBGetFunc,
) (*IFMIBCollection, error) {
	if sampledAt.IsZero() {
		sampledAt = time.Now()
	}

	if walk == nil {
		return nil, fmt.Errorf("IF-MIB walk function is required")
	}

	if get == nil {
		return nil, fmt.Errorf("IF-MIB get function is required")
	}

	specs := []ifMIBColumnSpec{
		{RootOID: IFNameOID, Required: true},
		{RootOID: IFDescrOID, Required: true},
		{RootOID: IFTypeOID, Required: false},
		{RootOID: IFAdminStatusOID, Required: true},
		{RootOID: IFOperStatusOID, Required: true},
		{RootOID: IFHighSpeedOID, Required: false},
		{RootOID: IFSpeedOID, Required: false},
		{RootOID: IFPhysAddressOID, Required: false},
		{RootOID: IFLastChangeOID, Required: false},
		{RootOID: IFHCInOctetsOID, Required: true},
		{RootOID: IFHCOutOctetsOID, Required: true},
		{RootOID: IFInErrorsOID, Required: false},
		{RootOID: IFOutErrorsOID, Required: false},
		{RootOID: IFInDiscardsOID, Required: false},
		{RootOID: IFOutDiscardsOID, Required: false},
	}

	columns := make([]IFMIBColumn, 0, len(specs))
	warnings := make([]string, 0)

	for _, spec := range specs {
		rows, err := walk(spec.RootOID)
		if err != nil {
			if spec.Required {
				return nil, fmt.Errorf(
					"required IF-MIB walk %s: %w",
					spec.RootOID,
					err,
				)
			}

			warnings = append(
				warnings,
				fmt.Sprintf(
					"optional IF-MIB walk %s unavailable: %v",
					spec.RootOID,
					err,
				),
			)

			continue
		}

		if spec.Required && len(rows) == 0 {
			return nil, fmt.Errorf(
				"required IF-MIB walk %s returned no rows",
				spec.RootOID,
			)
		}

		columns = append(
			columns,
			IFMIBColumn{
				RootOID: spec.RootOID,
				Rows:    rows,
			},
		)
	}

	sysUpTimeTicks := uint64(0)

	uptime, err := get(SysUpTimeOID)
	if err != nil {
		warnings = append(
			warnings,
			fmt.Sprintf(
				"optional sysUpTime unavailable: %v",
				err,
			),
		)
	} else {
		value, valueErr := Uint64Value(uptime.Value)
		if valueErr != nil {
			warnings = append(
				warnings,
				fmt.Sprintf(
					"optional sysUpTime invalid: %v",
					valueErr,
				),
			)
		} else {
			sysUpTimeTicks = value
		}
	}

	merged, err := MergeIFMIBColumns(columns)
	if err != nil {
		return nil, fmt.Errorf(
			"merge IF-MIB columns: %w",
			err,
		)
	}

	indexes := make([]int, 0, len(merged))

	for index := range merged {
		indexes = append(indexes, index)
	}

	sort.Ints(indexes)

	ports := make([]IFMIBPort, 0, len(indexes))

	for _, index := range indexes {
		port := merged[index]

		if port == nil {
			continue
		}

		ports = append(ports, *port)
	}

	return &IFMIBCollection{
		SampledAt:      sampledAt,
		SysUpTimeTicks: sysUpTimeTicks,
		Ports:          ports,
		Warnings:       warnings,
	}, nil
}
