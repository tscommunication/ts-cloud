import apiClient from "./client";

export interface OLTDashboardSummary {
  total_olts: number;
  online_olts: number;
  offline_olts: number;
  total_onus: number;
  online_onus: number;
  offline_onus: number;
  optical_missing: number;
}

export interface OLTDashboardOLT {
  id: number;
  code: string;
  name: string;
  vendor: string;
  olt_type: string;
  pop_id?: number | null;
  pop_name: string;
  monitoring_status: string;
  last_polled_at?: string | null;
  last_error: string;
  total_onus: number;
  online_onus: number;
  offline_onus: number;
  optical_missing: number;
}

export interface OLTDashboardVendor {
  vendor: string;
  olt_count: number;
  online_olts: number;
  total_onus: number;
  online_onus: number;
  offline_onus: number;
}

export interface OLTDashboardPOP {
  pop_id?: number | null;
  pop_name: string;
  olt_count: number;
  online_olts: number;
  total_onus: number;
  online_onus: number;
  offline_onus: number;
}

export interface OLTDashboard {
  summary: OLTDashboardSummary;
  olts: OLTDashboardOLT[];
  vendors: OLTDashboardVendor[];
  pops: OLTDashboardPOP[];
}

export async function getOLTDashboard(): Promise<OLTDashboard> {
  return (await apiClient.get<OLTDashboard>("/network/olt-dashboard")).data;
}
