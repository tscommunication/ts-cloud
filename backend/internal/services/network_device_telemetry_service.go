package services

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	snmpmonitor "github.com/tscommunication/ts-cloud/internal/monitoring/snmp"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func PersistNetworkDevicePortCandidates(
	networkDeviceID uint,
	candidates []snmpmonitor.PortPersistenceCandidate,
) error {
	if networkDeviceID == 0 {
		return errors.New("network device ID is required")
	}

	if len(candidates) == 0 {
		return errors.New("network device port candidates are required")
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		for index := range candidates {
			if err := persistNetworkDevicePortCandidateTx(
				tx,
				networkDeviceID,
				candidates[index],
			); err != nil {
				return fmt.Errorf(
					"persist port candidate %d: %w",
					index,
					err,
				)
			}
		}

		return nil
	})
}

const maxPostgresBigInt uint64 = 1<<63 - 1

func validatePortPersistenceCandidateDatabaseRange(
	candidate snmpmonitor.PortPersistenceCandidate,
) error {
	counters := []struct {
		name  string
		value uint64
	}{
		{name: "in_octets", value: candidate.InOctets},
		{name: "out_octets", value: candidate.OutOctets},
		{name: "in_errors", value: candidate.InErrors},
		{name: "out_errors", value: candidate.OutErrors},
		{name: "in_discards", value: candidate.InDiscards},
		{name: "out_discards", value: candidate.OutDiscards},
	}

	for _, counter := range counters {
		if counter.value > maxPostgresBigInt {
			return fmt.Errorf(
				"ifIndex %d counter %s exceeds signed BIGINT range",
				candidate.IfIndex,
				counter.name,
			)
		}
	}

	return nil
}

func persistNetworkDevicePortCandidateTx(
	tx *gorm.DB,
	networkDeviceID uint,
	candidate snmpmonitor.PortPersistenceCandidate,
) error {
	if tx == nil {
		return errors.New("database transaction is required")
	}

	if candidate.IfIndex <= 0 {
		return fmt.Errorf(
			"invalid ifIndex %d",
			candidate.IfIndex,
		)
	}

	if candidate.PortKey == "" {
		return errors.New("port key is required")
	}

	if candidate.SampledAt.IsZero() {
		return errors.New("sample time is required")
	}

	if err := validatePortPersistenceCandidateDatabaseRange(
		candidate,
	); err != nil {
		return err
	}

	ifIndex := candidate.IfIndex
	lastSeenAt := candidate.SampledAt

	port := models.NetworkDevicePort{
		NetworkDeviceID: networkDeviceID,
		PortKey:         candidate.PortKey,
		IfIndex:         &ifIndex,
		Name:            candidate.Name,
		Description:     candidate.Description,
		PortType:        candidate.PortType,
		AdminStatus:     candidate.AdminStatus,
		OperStatus:      candidate.OperStatus,
		SpeedMbps:       candidate.SpeedMbps,
		MACAddress:      candidate.MACAddress,
		LastChangeAt:    candidate.LastChangeAt,
		LastSeenAt:      &lastSeenAt,
		UpdatedAt:       candidate.SampledAt,
	}

	if err := repositories.UpsertNetworkDevicePortTx(
		tx,
		&port,
	); err != nil {
		return err
	}

	latest, err :=
		repositories.LatestNetworkDevicePortSampleTx(
			tx,
			port.ID,
		)
	if err != nil {
		return err
	}

	rate := snmpmonitor.PortSampleRateCandidate{}

	if latest != nil {
		if candidate.SampledAt.Before(latest.SampledAt) {
			return fmt.Errorf(
				"out-of-order sample: current=%s latest=%s",
				candidate.SampledAt.Format(time.RFC3339Nano),
				latest.SampledAt.Format(time.RFC3339Nano),
			)
		}

		if candidate.SampledAt.Equal(latest.SampledAt) {
			return nil
		}

		rate = snmpmonitor.BuildPortSampleRateCandidate(
			latest.SampledAt,
			latest.InOctets,
			latest.OutOctets,
			candidate,
		)
	}

	sample := models.NetworkDevicePortSample{
		NetworkDevicePortID: port.ID,
		SampledAt:           candidate.SampledAt,
		InOctets:            candidate.InOctets,
		OutOctets:           candidate.OutOctets,
		InMbps:              rate.InMbps,
		OutMbps:             rate.OutMbps,
		InErrors:            candidate.InErrors,
		OutErrors:           candidate.OutErrors,
		InDiscards:          candidate.InDiscards,
		OutDiscards:         candidate.OutDiscards,
	}

	return repositories.CreateNetworkDevicePortSampleTx(
		tx,
		&sample,
	)
}
