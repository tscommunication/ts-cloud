import apiClient from './client'

export interface CustomerImportPreview { total_rows: number; active_rows: number; inactive_rows: number; credential_rows: number; adoption_rows: number; packages: string[]; pops: string[]; warnings: string[] }
export interface CustomerImportBatch { id: number; filename: string; status: string; total_rows: number; imported_rows: number; created_packages: number; created_pops: number; created_agents: number; created_at: string }
const form = (file: File, routerId?: number) => { const data = new FormData(); data.append('file', file); if (routerId) data.append('router_id', String(routerId)); return data }
export async function previewCustomerImport(file: File): Promise<CustomerImportPreview> { return (await apiClient.post<CustomerImportPreview>('/customer-imports/preview', form(file))).data }
export async function importCustomers(file: File, routerId: number): Promise<CustomerImportBatch> { return (await apiClient.post<CustomerImportBatch>('/customer-imports', form(file, routerId))).data }
