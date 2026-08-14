import apiClient from './client'

export interface AgentDashboardSummary {
  total_customers: number
  active_customers: number
  active_subscriptions: number
  total_invoiced: number
  total_outstanding: number
  total_collected: number
  today_collected: number
  commission_earned: number
  commission_paid: number
  commission_payable: number
  overdue_invoices: number
	voided_collections: number
	voided_amount: number
}

export async function getAgentDashboard(): Promise<AgentDashboardSummary> {
  return (await apiClient.get<AgentDashboardSummary>('/agent-dashboard')).data
}
