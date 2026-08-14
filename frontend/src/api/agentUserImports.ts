import apiClient from './client'

export interface AgentUserImportRow {
  row_number: number
  name: string
  username: string
  role: string
  active: boolean
  agent_id?: number
  agent_name?: string
  status: 'READY_CREATE' | 'READY_UPDATE' | 'SKIPPED_UNMATCHED_AGENT'
}

export interface AgentUserImportPreview {
  total_rows: number
  ready_rows: number
  create_rows: number
  update_rows: number
  skipped_rows: number
  rows: AgentUserImportRow[]
  warnings: string[]
}

export interface AgentUserImportResult {
  total_rows: number
  created_rows: number
  updated_rows: number
  skipped_rows: number
}

const form = (file: File) => {
  const data = new FormData()
  data.append('file', file)
  return data
}

export async function previewAgentUserImport(file: File): Promise<AgentUserImportPreview> {
  return (await apiClient.post<AgentUserImportPreview>('/agent-user-imports/preview', form(file))).data
}

export async function importAgentUsers(file: File): Promise<AgentUserImportResult> {
  return (await apiClient.post<AgentUserImportResult>('/agent-user-imports', form(file))).data
}
