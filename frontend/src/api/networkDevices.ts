import apiClient from "./client";

export interface NetworkDevice {
  id: number; code: string; name: string; device_type: "OLT" | "SWITCH" | "MIKROTIK";
  vendor: string; model: string; olt_type: string; pop_id?: number; pop_name: string;
  management_ip: string; management_port: number; router_ids: number[]; router_names: string[];
  monitoring_protocol: "SNMP" | "MIKROTIK_API"; snmp_version: "V2C" | "V3" | "";
  snmp_port: number; snmp_username: string; snmp_community: string; credential_configured: boolean;
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
