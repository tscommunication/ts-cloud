package snmp

import (
	"strings"
	"time"
)

type ONUOpticalRecord struct {
	PONNo        int
	ONUNo        int
	TemperatureC *float64
	VoltageV     *float64
	TxBiasMA     *float64
	TxPowerDBM   *float64
	RxPowerDBM   *float64
}

type ONUOpticalCollection struct {
	Vendor    string
	SampledAt time.Time
	Records   []ONUOpticalRecord
}

type ONUVendorAdapter interface {
	Name() string
	Matches(vendor string, sysObjectID string) bool
	CollectOptical(
		cfg V2CConfig,
		sampledAt time.Time,
	) (*ONUOpticalCollection, error)
	BuildPersistenceCandidates(
		ifmib *IFMIBCollection,
		optical *ONUOpticalCollection,
	) ([]ONUPersistenceCandidate, error)
}

type ONURegistrationTimeCollector interface {
	CollectRegistrationTimes(
		cfg V2CConfig,
		location *time.Location,
	) ([]VSOLONURegistrationRecord, error)
}

func ResolveONUVendorAdapter(
	vendor string,
	sysObjectID string,
) (ONUVendorAdapter, bool) {
	adapters := []ONUVendorAdapter{
		VSOLONUAdapter{},
		BDCOMONUAdapter{},
		ECOMONUAdapter{},
	}

	for _, adapter := range adapters {
		if adapter.Matches(vendor, sysObjectID) {
			return adapter, true
		}
	}

	return nil, false
}

func normalizedVendor(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
