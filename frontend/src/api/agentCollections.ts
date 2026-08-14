import apiClient from './client'

export interface AgentCollection {
  id: number; agent_id: number; agent_name: string; customer_id: number; customer_code: string; customer_name: string
  payment_id: number; receipt_no: string; amount: number; commission_rate: number; commission_amount: number
  status: 'ACTIVE' | 'VOID'; collected_at: string
}
export interface AgentCollectionReport { collections: AgentCollection[]; count: number; total_amount: number; total_commission: number; void_count: number; void_amount: number }

export async function getAgentCollections(params: { agent_id?: number; status?: '' | 'ACTIVE' | 'VOID' } = {}): Promise<AgentCollectionReport> {
  return (await apiClient.get<AgentCollectionReport>('/agent-collections', { params })).data
}
