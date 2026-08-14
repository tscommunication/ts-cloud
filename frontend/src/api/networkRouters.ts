import apiClient from './client'

export type RouterStatus = 'ACTIVE' | 'INACTIVE' | 'MAINTENANCE'
export interface NetworkRouter { id: number; code: string; name: string; pop_id?: number; pop_name: string; host: string; api_port: number; api_username: string; use_tls: boolean; status: RouterStatus; remarks: string; connectivity_status: 'UNKNOWN' | 'ONLINE' | 'OFFLINE'; last_checked_at?: string; last_latency_ms: number; last_connection_error: string; credentials_configured: boolean }
export interface NetworkRouterInput { code: string; name: string; pop_id?: number; host: string; api_port: number; api_username: string; use_tls: boolean; status: RouterStatus; remarks: string }

export async function getNetworkRouters(): Promise<NetworkRouter[]> { return (await apiClient.get<{ routers: NetworkRouter[] }>('/network/routers')).data.routers }
export async function createNetworkRouter(data: NetworkRouterInput): Promise<NetworkRouter> { return (await apiClient.post<NetworkRouter>('/network/routers', data)).data }
export async function updateNetworkRouter(id: number, data: NetworkRouterInput): Promise<NetworkRouter> { return (await apiClient.put<NetworkRouter>(`/network/routers/${id}`, data)).data }
export async function testNetworkRouterConnection(id: number): Promise<NetworkRouter> { return (await apiClient.post<NetworkRouter>(`/network/routers/${id}/test-connection`)).data }
export async function setNetworkRouterCredentials(id: number, password: string): Promise<NetworkRouter> { return (await apiClient.put<NetworkRouter>(`/network/routers/${id}/credentials`, { password })).data }
