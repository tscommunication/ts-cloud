package models

import "time"

type NetworkDevicePort struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	NetworkDeviceID uint           `gorm:"not null;index;uniqueIndex:idx_network_device_port_key,priority:1" json:"network_device_id"`
	NetworkDevice   *NetworkDevice `gorm:"foreignKey:NetworkDeviceID;constraint:OnDelete:CASCADE" json:"-"`
	PortKey         string         `gorm:"size:120;not null;uniqueIndex:idx_network_device_port_key,priority:2" json:"port_key"`
	IfIndex         *int           `gorm:"index" json:"if_index"`
	VendorPortRef   string         `gorm:"size:120;index" json:"vendor_port_ref"`
	Name            string         `gorm:"size:160" json:"name"`
	Description     string         `gorm:"size:255" json:"description"`
	PortType        string         `gorm:"size:40;index" json:"port_type"`
	AdminStatus     string         `gorm:"size:20;index" json:"admin_status"`
	OperStatus      string         `gorm:"size:20;index" json:"oper_status"`
	SpeedMbps       int64          `gorm:"not null;default:0" json:"speed_mbps"`
	MACAddress      string         `gorm:"size:32;index" json:"mac_address"`
	LastChangeAt    *time.Time     `json:"last_change_at"`
	LastSeenAt      *time.Time     `gorm:"index" json:"last_seen_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type NetworkDevicePortSample struct {
	ID                  uint               `gorm:"primaryKey" json:"id"`
	NetworkDevicePortID uint               `gorm:"not null;index:idx_network_device_port_sampled,priority:1" json:"network_device_port_id"`
	NetworkDevicePort   *NetworkDevicePort `gorm:"foreignKey:NetworkDevicePortID;constraint:OnDelete:CASCADE" json:"-"`
	SampledAt           time.Time          `gorm:"not null;index:idx_network_device_port_sampled,priority:2" json:"sampled_at"`
	InOctets            uint64             `gorm:"not null;default:0" json:"in_octets"`
	OutOctets           uint64             `gorm:"not null;default:0" json:"out_octets"`
	InMbps              float64            `gorm:"not null;default:0" json:"in_mbps"`
	OutMbps             float64            `gorm:"not null;default:0" json:"out_mbps"`
	InErrors            uint64             `gorm:"not null;default:0" json:"in_errors"`
	OutErrors           uint64             `gorm:"not null;default:0" json:"out_errors"`
	InDiscards          uint64             `gorm:"not null;default:0" json:"in_discards"`
	OutDiscards         uint64             `gorm:"not null;default:0" json:"out_discards"`
}

type NetworkDeviceONU struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	NetworkDeviceID    uint           `gorm:"not null;index;uniqueIndex:idx_network_device_onu_key,priority:1" json:"network_device_id"`
	NetworkDevice      *NetworkDevice `gorm:"foreignKey:NetworkDeviceID;constraint:OnDelete:CASCADE" json:"-"`
	PONNo              int            `gorm:"not null;uniqueIndex:idx_network_device_onu_key,priority:2" json:"pon_no"`
	ONUNo              int            `gorm:"not null;uniqueIndex:idx_network_device_onu_key,priority:3" json:"onu_no"`
	IfIndex            *int           `gorm:"index" json:"if_index"`
	MACAddress         string         `gorm:"size:32;index" json:"mac_address"`
	SerialNumber       string         `gorm:"size:120;index" json:"serial_number"`
	Model              string         `gorm:"size:120" json:"model"`
	Capability         string         `gorm:"size:120" json:"capability"`
	Description        string         `gorm:"size:255" json:"description"`
	OperStatus         string         `gorm:"size:20;index" json:"oper_status"`
	DistanceM          int            `gorm:"not null;default:0" json:"distance_m"`
	LastRegisteredAt   *time.Time     `json:"last_registered_at"`
	LastDeregisteredAt *time.Time     `json:"last_deregistered_at"`
	UptimeSeconds      int64          `gorm:"not null;default:0" json:"uptime_seconds"`
	LastSeenAt         *time.Time     `gorm:"index" json:"last_seen_at"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

type NetworkDeviceONUSample struct {
	ID                 uint              `gorm:"primaryKey" json:"id"`
	NetworkDeviceONUID uint              `gorm:"column:network_device_onu_id;not null;index:idx_network_device_onu_sampled,priority:1" json:"network_device_onu_id"`
	NetworkDeviceONU   *NetworkDeviceONU `gorm:"foreignKey:NetworkDeviceONUID;constraint:OnDelete:CASCADE" json:"-"`
	SampledAt          time.Time         `gorm:"not null;index:idx_network_device_onu_sampled,priority:2" json:"sampled_at"`
	InOctets           uint64            `gorm:"not null;default:0" json:"in_octets"`
	OutOctets          uint64            `gorm:"not null;default:0" json:"out_octets"`
	InMbps             float64           `gorm:"not null;default:0" json:"in_mbps"`
	OutMbps            float64           `gorm:"not null;default:0" json:"out_mbps"`
	TemperatureC       *float64          `json:"temperature_c"`
	VoltageV           *float64          `json:"voltage_v"`
	TxBiasMA           *float64          `json:"tx_bias_ma"`
	TxPowerDBM         *float64          `json:"tx_power_dbm"`
	RxPowerDBM         *float64          `json:"rx_power_dbm"`
	DistanceM          *int              `json:"distance_m"`
}
