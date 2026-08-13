import apiClient from './client'

export interface BillingSummary {
  total_invoiced: number
  total_collected: number
  total_outstanding: number
  today_collected: number
  overdue_invoices: number
  unpaid_invoices: number
}

export interface BillingRun {
  ID: number
  run_date: string
  triggered_by: number
  status: string
  total: number
  created_count: number
  skipped_count: number
  failed_count: number
}

export async function getBillingSummary(): Promise<BillingSummary> {
  return (await apiClient.get<BillingSummary>('/billing/summary')).data
}

export async function getBillingRuns(): Promise<BillingRun[]> {
  return (await apiClient.get<BillingRun[]>('/billing/runs')).data
}

export async function runBilling(): Promise<BillingRun> {
  return (await apiClient.post<BillingRun>('/billing/run')).data
}
