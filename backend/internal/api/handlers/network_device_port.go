package handlers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/services"
)

type networkDevicePortSampleResponse struct {
	SampledAt   time.Time `json:"sampled_at"`
	InOctets    uint64    `json:"in_octets"`
	OutOctets   uint64    `json:"out_octets"`
	InMbps      float64   `json:"in_mbps"`
	OutMbps     float64   `json:"out_mbps"`
	InErrors    uint64    `json:"in_errors"`
	OutErrors   uint64    `json:"out_errors"`
	InDiscards  uint64    `json:"in_discards"`
	OutDiscards uint64    `json:"out_discards"`
}

type networkDevicePortResponse struct {
	ID            uint                             `json:"id"`
	PortKey       string                           `json:"port_key"`
	IfIndex       *int                             `json:"if_index"`
	VendorPortRef string                           `json:"vendor_port_ref"`
	Name          string                           `json:"name"`
	Description   string                           `json:"description"`
	PortType      string                           `json:"port_type"`
	AdminStatus   string                           `json:"admin_status"`
	OperStatus    string                           `json:"oper_status"`
	SpeedMbps     int64                            `json:"speed_mbps"`
	MACAddress    string                           `json:"mac_address"`
	LastChangeAt  *time.Time                       `json:"last_change_at"`
	LastSeenAt    *time.Time                       `json:"last_seen_at"`
	LatestSample  *networkDevicePortSampleResponse `json:"latest_sample"`
}

func ListNetworkDevicePorts(c *gin.Context) {
	id, err := strconv.ParseUint(
		c.Param("id"),
		10,
		64,
	)
	if err != nil || id == 0 {
		c.JSON(
			400,
			gin.H{"error": "Invalid device ID"},
		)
		return
	}

	views, err := services.ListNetworkDevicePortViews(
		uint(id),
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(
			404,
			gin.H{"error": "Device not found"},
		)
		return
	}

	if err != nil {
		c.JSON(
			500,
			gin.H{
				"error": "Failed to load network device ports",
			},
		)
		return
	}

	result := make(
		[]networkDevicePortResponse,
		0,
		len(views),
	)

	for _, view := range views {
		port := view.Port

		item := networkDevicePortResponse{
			ID:            port.ID,
			PortKey:       port.PortKey,
			IfIndex:       port.IfIndex,
			VendorPortRef: port.VendorPortRef,
			Name:          port.Name,
			Description:   port.Description,
			PortType:      port.PortType,
			AdminStatus:   port.AdminStatus,
			OperStatus:    port.OperStatus,
			SpeedMbps:     port.SpeedMbps,
			MACAddress:    port.MACAddress,
			LastChangeAt:  port.LastChangeAt,
			LastSeenAt:    port.LastSeenAt,
		}

		if view.LatestSample != nil {
			sample := view.LatestSample

			item.LatestSample =
				&networkDevicePortSampleResponse{
					SampledAt:   sample.SampledAt,
					InOctets:    sample.InOctets,
					OutOctets:   sample.OutOctets,
					InMbps:      sample.InMbps,
					OutMbps:     sample.OutMbps,
					InErrors:    sample.InErrors,
					OutErrors:   sample.OutErrors,
					InDiscards:  sample.InDiscards,
					OutDiscards: sample.OutDiscards,
				}
		}

		result = append(result, item)
	}

	c.JSON(
		200,
		gin.H{"ports": result},
	)
}
