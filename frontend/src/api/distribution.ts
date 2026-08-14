import apiClient from './client'

export interface POP { id: number; code: string; name: string; manager_name: string; mobile: string; address: string; status: 'ACTIVE' | 'INACTIVE' }
export interface Agent { id: number; code: string; name: string; pop_id: number; pop_name: string; pop_ids: number[]; pop_names: string[]; mobile: string; address: string; commission_percent: number; opening_balance: number; source_reference: string; status: 'ACTIVE' | 'INACTIVE' }
export type POPInput = Omit<POP, 'id' | 'status'>
export type AgentInput = Omit<Agent, 'id' | 'status' | 'pop_name' | 'pop_names' | 'opening_balance' | 'source_reference'>

export async function getPOPs(): Promise<POP[]> { return (await apiClient.get<{ pops: POP[] }>('/pops')).data.pops }
export async function createPOP(data: POPInput): Promise<POP> { return (await apiClient.post<POP>('/pops', data)).data }
export async function updatePOP(id: number, data: Omit<POPInput, 'code'>): Promise<POP> { return (await apiClient.put<POP>(`/pops/${id}`, data)).data }
export async function setPOPStatus(id: number, status: POP['status']): Promise<POP> { return (await apiClient.patch<POP>(`/pops/${id}/status`, { status })).data }
export async function getAgents(popId?: number): Promise<Agent[]> { return (await apiClient.get<{ agents: Agent[] }>('/agents', { params: popId ? { pop_id: popId } : undefined })).data.agents }
export async function createAgent(data: AgentInput): Promise<Agent> { return (await apiClient.post<Agent>('/agents', data)).data }
export async function updateAgent(id: number, data: Omit<AgentInput, 'code'>): Promise<Agent> { return (await apiClient.put<Agent>(`/agents/${id}`, data)).data }
export async function setAgentStatus(id: number, status: Agent['status']): Promise<Agent> { return (await apiClient.patch<Agent>(`/agents/${id}/status`, { status })).data }
