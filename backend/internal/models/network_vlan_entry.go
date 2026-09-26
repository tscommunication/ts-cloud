package models

import "gorm.io/gorm"

type NetworkVLANEntry struct {
	gorm.Model
	DeviceType string `gorm:"size:10;not null;uniqueIndex:idx_network_vlan_entry_scope,priority:1" json:"device_type"`
	DeviceID   uint   `gorm:"not null;uniqueIndex:idx_network_vlan_entry_scope,priority:2" json:"device_id"`
	PortName   string `gorm:"size:120;not null;uniqueIndex:idx_network_vlan_entry_scope,priority:3" json:"port_name"`
	VLANID     int    `gorm:"not null;uniqueIndex:idx_network_vlan_entry_scope,priority:4" json:"vlan_id"`
	VLANName   string `gorm:"size:120;not null" json:"vlan_name"`
}
