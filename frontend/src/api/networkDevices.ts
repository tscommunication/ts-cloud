import apiClient from "./client";

export interface NetworkDevice {
  id: number; code: string; name: string; device_type: "OLT" | "SWITCH" | "MIKROTIK";
  vendor: string; model: string; olt_type: string; pop_id?: number; pop_name: string;
  management_ip: string; management_port: number; router_ids: number[]; router_names: string[];
  monitoring_protocol: "SNMP" | "MIKROTIK_API"; snmp_version: "V2C" | "V3" | "";
  snmp_port: number; snmp_username: string; credential_configured: boolean;
  polling_interval_seconds: number; monitoring_enabled: boolean; monitoring_status: string;
  last_polled_at?: string; last_error: string; remarks: string;
}

export interface NetworkDeviceInput {
  code: string; name: string; device_type: string; vendor: string; model: string;
  olt_type: string; pop_id?: number; management_ip: string; management_port: number;
  router_ids: number[]; monitoring_protocol: string; snmp_version: string; snmp_port: number;
  snmp_username: string; snmp_secret?: string; polling_interval_seconds: number;
  monitoring_enabled: boolean; remarks: string;
}

export async function getNetworkDevices(): Promise<NetworkDevice[]> {
  return (await apiClient.get<{ devices: NetworkDevice[] | null }>("/network/devices")).data.devices ?? [];
}
export async function createNetworkDevice(data: NetworkDeviceInput): Promise<NetworkDevice> {
  return (await apiClient.post("/network/devices", data)).data;
}
export async function updateNetworkDevice(id: number, data: NetworkDeviceInput): Promise<NetworkDevice> {
  return (await apiClient.put(`/network/devices/${id}`, data)).data;
}
export async function deleteNetworkDevice(id: number): Promise<void> {
  await apiClient.delete(`/network/devices/${id}`);
}
export async function testNetworkDeviceConnection(id: number): Promise<NetworkDevice> {
  return (await apiClient.post(`/network/devices/${id}/test-connection`)).data;
}

export interface NetworkDevicePortSample {
  sampled_at: string;
  in_octets: number;
  out_octets: number;
  in_mbps: number;
  out_mbps: number;
  in_errors: number;
  out_errors: number;
  in_discards: number;
  out_discards: number;
}

export interface NetworkDevicePort {
  id: number;
  port_key: string;
  if_index?: number;
  vendor_port_ref: string;
  name: string;
  description: string;
  port_type: string;
  admin_status: string;
  oper_status: string;
  speed_mbps: number;
  mac_address: string;
  last_change_at?: string;
  last_seen_at?: string;
  latest_sample?: NetworkDevicePortSample | null;
}

export async function getNetworkDevicePorts(
  id: number,
): Promise<NetworkDevicePort[]> {
  const response = await apiClient.get<{
    ports: NetworkDevicePort[] | null;
  }>(`/network/devices/${id}/ports`);

  return response.data.ports ?? [];
}

export interface NetworkDeviceONUSample {
  in_mbps: number;
  out_mbps: number;
  temperature_c?: number | null;
  voltage_v?: number | null;
  tx_power_dbm?: number | null;
  rx_power_dbm?: number | null;
  distance_m?: number | null;
}

export interface NetworkDeviceONUOpticalSample {
  sampled_at: string;
  temperature_c?: number | null;
  voltage_v?: number | null;
  tx_bias_ma?: number | null;
  tx_power_dbm?: number | null;
  rx_power_dbm?: number | null;
}

export interface NetworkDeviceONU {
  id: number;
  network_device_id: number;
  pon_no: number;
  onu_no: number;
  if_index?: number | null;
  mac_address: string;
  serial_number: string;
  model: string;
  capability: string;
  description: string;
  oper_status: string;
  last_deregistered_at?: string | null;
  distance_m: number;
  latest_sample?: NetworkDeviceONUSample | null;
  latest_optical?: NetworkDeviceONUOpticalSample | null;
}

export async function getNetworkDeviceONUs(
  id: number,
): Promise<NetworkDeviceONU[]> {
  const response = await apiClient.get<{
    onus: NetworkDeviceONU[] | null;
  }>(`/network/devices/${id}/onus`);

  return response.data.onus ?? [];
}
