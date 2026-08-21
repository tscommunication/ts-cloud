import apiClient from './client'

export interface POP {
  id: number
  code: string
  name: string
  manager_name: string
  mobile: string
  address: string
  source_reference: string
  status: 'ACTIVE' | 'INACTIVE'
  deleted_at?: string
}

export interface Agent {
  id: number
  code: string
  name: string
  pop_id: number
  pop_name: string
  pop_ids: number[]
  pop_names: string[]
  package_ids: number[]
  package_names: string[]
  router_ids: number[]
  router_names: string[]
  mobile: string
  address: string
  commission_percent: number
  opening_balance: number
  source_reference: string
  status: 'ACTIVE' | 'INACTIVE'
  deleted_at?: string
}

export type POPInput = Omit<
  POP,
  'id' | 'status' | 'source_reference' | 'deleted_at'
>

export type AgentInput = Omit<
  Agent,
  | 'id'
  | 'status'
  | 'pop_name'
  | 'pop_names'
  | 'package_names'
  | 'router_names'
  | 'opening_balance'
  | 'source_reference'
  | 'deleted_at'
>

export interface POPMigrationResult {
  source_pop_id: number
  target_pop_id: number
  customers_migrated: number
  primary_agents_migrated: number
  agent_memberships_migrated: number
  routers_migrated: number
}

export interface AgentMigrationResult {
  source_agent_id: number
  target_agent_id: number
  customers_migrated: number
  login_users_migrated: number
  pops_migrated: number
}

export async function getPOPs(): Promise<POP[]> {
  const response = await apiClient.get<{ pops: POP[] | null }>('/pops')
  return response.data.pops ?? []
}

export async function getArchivedPOPs(): Promise<POP[]> {
  const response = await apiClient.get<{ pops: POP[] }>(
    '/pops/archived',
  )

  return response.data.pops
}

export async function restorePOP(id: number): Promise<POP> {
  const response = await apiClient.post<POP>(
    `/pops/${id}/restore`,
  )

  return response.data
}

export async function createPOP(data: POPInput): Promise<POP> {
  const response = await apiClient.post<POP>('/pops', data)
  return response.data
}

export async function updatePOP(
  id: number,
  data: Omit<POPInput, 'code'>,
): Promise<POP> {
  const response = await apiClient.put<POP>(`/pops/${id}`, data)
  return response.data
}

export async function setPOPStatus(
  id: number,
  status: POP['status'],
): Promise<POP> {
  const response = await apiClient.patch<POP>(
    `/pops/${id}/status`,
    { status },
  )
  return response.data
}

export async function migratePOP(
  id: number,
  targetPOPID: number,
): Promise<POPMigrationResult> {
  const response = await apiClient.post<POPMigrationResult>(
    `/pops/${id}/migrate`,
    {
      target_pop_id: targetPOPID,
    },
  )

  return response.data
}

export async function deletePOP(id: number): Promise<void> {
  await apiClient.delete(`/pops/${id}`)
}

export async function getAgents(
  popId?: number,
): Promise<Agent[]> {
  const response = await apiClient.get<{ agents: Agent[] }>(
    '/agents',
    {
      params: popId ? { pop_id: popId } : undefined,
    },
  )

  return response.data.agents
}

export async function getArchivedAgents(): Promise<Agent[]> {
  const response = await apiClient.get<{ agents: Agent[] }>(
    '/agents/archived',
  )

  return response.data.agents
}

export async function restoreAgent(
  id: number,
): Promise<Agent> {
  const response = await apiClient.post<Agent>(
    `/agents/${id}/restore`,
  )

  return response.data
}

export async function createAgent(
  data: AgentInput,
): Promise<Agent> {
  const response = await apiClient.post<Agent>('/agents', data)
  return response.data
}

export async function updateAgent(
  id: number,
  data: Omit<AgentInput, 'code'>,
): Promise<Agent> {
  const response = await apiClient.put<Agent>(
    `/agents/${id}`,
    data,
  )

  return response.data
}

export async function updateAgentPackages(
  id: number,
  packageIds: number[],
): Promise<Agent> {
  const response = await apiClient.put<Agent>(`/agents/${id}/packages`, {
    package_ids: packageIds,
  })
  return response.data
}

export async function updateAgentPermissions(id: number, packageIds: number[], routerIds: number[]): Promise<Agent> {
  return (await apiClient.put<Agent>(`/agents/${id}/permissions`, { package_ids: packageIds, router_ids: routerIds })).data
}

export async function setAgentStatus(
  id: number,
  status: Agent['status'],
): Promise<Agent> {
  const response = await apiClient.patch<Agent>(
    `/agents/${id}/status`,
    { status },
  )

  return response.data
}

export async function migrateAgent(
  id: number,
  targetAgentId: number,
): Promise<AgentMigrationResult> {
  const response = await apiClient.post<AgentMigrationResult>(
    `/agents/${id}/migrate`,
    {
      target_agent_id: targetAgentId,
    },
  )

  return response.data
}

export async function deleteAgent(id: number): Promise<void> {
  await apiClient.delete(`/agents/${id}`)
}

export async function updateManagedCode(
  entity: 'agent' | 'pop' | 'package',
  id: number,
  code: string,
  reason: string,
): Promise<void> {
  await apiClient.put(`/code-management/${entity}/${id}`, { code, reason })
}
