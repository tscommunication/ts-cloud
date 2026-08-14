import apiClient from './client'

export interface AgentSettlement { id: number; settlement_no: string; agent_id: number; agent_name: string; amount: number; method: string; transaction_id: string; paid_at: string; status: 'PAID' | 'VOID'; remarks: string }
export interface AgentSettlementReport { settlements: AgentSettlement[]; earned: number; paid: number; payable: number }
export interface CreateAgentSettlement { agent_id: number; amount: number; method: string; transaction_id: string; paid_at: string; remarks: string }
export async function getAgentSettlements(agentId: number): Promise<AgentSettlementReport> { return (await apiClient.get<AgentSettlementReport>('/agent-settlements', { params: { agent_id: agentId } })).data }
export async function createAgentSettlement(data: CreateAgentSettlement): Promise<AgentSettlement> { return (await apiClient.post<AgentSettlement>('/agent-settlements', data)).data }
export async function voidAgentSettlement(id: number): Promise<void> { await apiClient.post(`/agent-settlements/${id}/void`) }
