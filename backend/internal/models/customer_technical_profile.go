package models

import (
	"time"

	"gorm.io/gorm"
)

type CustomerTechnicalProfile struct {
	ID uint `gorm:"primaryKey" json:"id"`

	CustomerID uint     `gorm:"not null;uniqueIndex" json:"customer_id"`
	Customer   Customer `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`

	ONUMAC               string `gorm:"column:onu_mac;size:100" json:"onu_mac"`
	OLTPON               string `gorm:"column:olt_pon;size:100" json:"olt_pon"`
	OLTSlot              string `gorm:"column:olt_slot;size:50" json:"olt_slot"`
	OLTPort              string `gorm:"column:olt_port;size:50" json:"olt_port"`
	ONUType              string `gorm:"column:onu_type;size:100" json:"onu_type"`
	ONUModel             string `gorm:"column:onu_model;size:150" json:"onu_model"`
	ONUIP                string `gorm:"column:onu_ip;size:100" json:"onu_ip"`
	ONUPasswordEncrypted string `gorm:"column:onu_password_encrypted;type:text" json:"-"`
	ONUSerial            string `gorm:"column:onu_serial;size:150" json:"onu_serial"`
	ONUSN                string `gorm:"column:onu_sn;size:150" json:"onu_sn"`

	RouterBrand             string `gorm:"size:100" json:"router_brand"`
	RouterModel             string `gorm:"size:150" json:"router_model"`
	RouterIP                string `gorm:"column:router_ip;size:100" json:"router_ip"`
	MikroTikPort            string `gorm:"column:mikrotik_port;size:120" json:"mikrotik_port"`
	RouterPasswordEncrypted string `gorm:"column:router_password_encrypted;type:text" json:"-"`

	CableType   string  `gorm:"size:100" json:"cable_type"`
	CableLength float64 `gorm:"default:0" json:"cable_length"`

	MediaConverterMAC               string `gorm:"column:media_converter_mac;size:100" json:"media_converter_mac"`
	MediaConverterIP                string `gorm:"column:media_converter_ip;size:100" json:"media_converter_ip"`
	MediaConverterPasswordEncrypted string `gorm:"column:media_converter_password_encrypted;type:text" json:"-"`

	SwitchModel             string `gorm:"size:150" json:"switch_model"`
	SwitchPort              string `gorm:"size:100" json:"switch_port"`
	SwitchIP                string `gorm:"column:switch_ip;size:100" json:"switch_ip"`
	SwitchPasswordEncrypted string `gorm:"column:switch_password_encrypted;type:text" json:"-"`

	AdditionalNote string `gorm:"type:text" json:"additional_note"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
