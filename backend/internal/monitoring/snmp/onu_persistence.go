package snmp

import "time"

type ONUPersistenceCandidate struct {
	PONNo   int
	ONUNo   int
	IfIndex int

	Description string
	OperStatus  string

	InOctets  uint64
	OutOctets uint64

	TemperatureC *float64
	VoltageV     *float64
	TxBiasMA     *float64
	TxPowerDBM   *float64
	RxPowerDBM   *float64

	SampledAt time.Time
}
