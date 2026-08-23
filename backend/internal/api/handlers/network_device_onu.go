package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/services"
)

func ListNetworkDeviceONUs(c *gin.Context) {
	id, err := strconv.ParseUint(
		c.Param("id"),
		10,
		64,
	)
	if err != nil || id == 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Invalid device ID"},
		)
		return
	}

	views, err := services.ListNetworkDeviceONUViews(
		uint(id),
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "Device not found"},
		)
		return
	}
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to load network device ONUs"},
		)
		return
	}

	result := make(
		[]gin.H,
		0,
		len(views),
	)

	for _, view := range views {
		onu := view.ONU

		item := gin.H{
			"id":                   onu.ID,
			"network_device_id":    onu.NetworkDeviceID,
			"pon_no":               onu.PONNo,
			"onu_no":               onu.ONUNo,
			"if_index":             onu.IfIndex,
			"mac_address":          onu.MACAddress,
			"serial_number":        onu.SerialNumber,
			"model":                onu.Model,
			"capability":           onu.Capability,
			"description":          onu.Description,
			"oper_status":          onu.OperStatus,
			"last_deregistered_at": onu.LastDeregisteredAt,
			"distance_m":           onu.DistanceM,
			"latest_sample":        nil,
			"latest_optical":       nil,
		}

		if view.LatestSample != nil {
			sample := view.LatestSample

			distanceM := onu.DistanceM
			if sample.DistanceM != nil {
				distanceM = *sample.DistanceM
			}

			item["latest_sample"] = gin.H{
				"in_mbps":       sample.InMbps,
				"out_mbps":      sample.OutMbps,
				"temperature_c": sample.TemperatureC,
				"voltage_v":     sample.VoltageV,
				"tx_power_dbm":  sample.TxPowerDBM,
				"rx_power_dbm":  sample.RxPowerDBM,
				"distance_m":    distanceM,
			}
		}

		if view.LatestOptical != nil {
			optical := view.LatestOptical

			item["latest_optical"] = gin.H{
				"sampled_at":    optical.SampledAt,
				"temperature_c": optical.TemperatureC,
				"voltage_v":     optical.VoltageV,
				"tx_bias_ma":    optical.TxBiasMA,
				"tx_power_dbm":  optical.TxPowerDBM,
				"rx_power_dbm":  optical.RxPowerDBM,
			}
		}

		result = append(result, item)
	}

	c.JSON(
		http.StatusOK,
		gin.H{"onus": result},
	)
}
