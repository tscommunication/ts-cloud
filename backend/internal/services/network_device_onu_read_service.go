package services

import (
	"errors"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

type NetworkDeviceONUView struct {
	ONU           models.NetworkDeviceONU
	LatestSample  *models.NetworkDeviceONUSample
	LatestOptical *models.NetworkDeviceONUSample
}

func ListNetworkDeviceONUViews(
	deviceID uint,
) ([]NetworkDeviceONUView, error) {
	if deviceID == 0 {
		return nil, errors.New(
			"network device ID is required",
		)
	}

	if _, err := GetNetworkDevice(deviceID); err != nil {
		return nil, err
	}

	onus, err := repositories.ListNetworkDeviceONUs(
		deviceID,
	)
	if err != nil {
		return nil, err
	}

	result := make(
		[]NetworkDeviceONUView,
		0,
		len(onus),
	)

	for _, onu := range onus {
		sample, err :=
			repositories.LatestNetworkDeviceONUSample(
				onu.ID,
			)
		if err != nil {
			return nil, err
		}

		optical, err :=
			repositories.LatestNetworkDeviceONUOpticalSample(
				onu.ID,
			)
		if err != nil {
			return nil, err
		}

		result = append(
			result,
			NetworkDeviceONUView{
				ONU:           onu,
				LatestSample:  sample,
				LatestOptical: optical,
			},
		)
	}

	return result, nil
}
