package services

import (
	"errors"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

type NetworkDevicePortView struct {
	Port         models.NetworkDevicePort
	LatestSample *models.NetworkDevicePortSample
}

func ListNetworkDevicePortViews(
	deviceID uint,
) ([]NetworkDevicePortView, error) {
	if deviceID == 0 {
		return nil, errors.New(
			"network device ID is required",
		)
	}

	if _, err := GetNetworkDevice(deviceID); err != nil {
		return nil, err
	}

	ports, err := repositories.ListNetworkDevicePorts(
		deviceID,
	)
	if err != nil {
		return nil, err
	}

	result := make(
		[]NetworkDevicePortView,
		0,
		len(ports),
	)

	for _, port := range ports {
		sample, err :=
			repositories.LatestNetworkDevicePortSample(
				port.ID,
			)
		if err != nil {
			return nil, err
		}

		result = append(
			result,
			NetworkDevicePortView{
				Port:         port,
				LatestSample: sample,
			},
		)
	}

	return result, nil
}
