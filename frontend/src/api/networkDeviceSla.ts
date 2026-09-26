import apiClient from './client'

export interface NetworkDeviceSLA {
  id: number
  code: string
  name: string
  device_type: string
  status: string
  samples: number
  observed_seconds: number
  downtime_seconds: number
  uptime_percent: number
}

export async function getNetworkDeviceSLA(days: 1 | 7 | 30): Promise<NetworkDeviceSLA[]> {
  return (await apiClient.get<{ devices: NetworkDeviceSLA[] }>('/network/device-sla', { params: { days } })).data.devices ?? []
}
