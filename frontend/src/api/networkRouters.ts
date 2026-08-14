import apiClient from './client'

export type RouterStatus = 'ACTIVE' | 'INACTIVE' | 'MAINTENANCE'
export interface NetworkRouter { id: number; code: string; name: string; pop_id?: number; pop_name: string; host: string; api_port: number; api_username: string; use_tls: boolean; status: RouterStatus; remarks: string; connectivity_status: 'UNKNOWN' | 'ONLINE' | 'OFFLINE'; last_checked_at?: string; last_latency_ms: number; last_connection_error: string; last_tcp_error: string; last_api_error: string; credentials_configured: boolean; api_status: 'UNKNOWN' | 'AUTHENTICATED' | 'AUTH_FAILED'; last_authenticated_at?: string; router_identity: string; routeros_version: string; board_name: string; router_uptime: string; cpu_load: number; total_memory: number; free_memory: number }
export interface NetworkRouterInput { code: string; name: string; pop_id?: number; host: string; api_port: number; api_username: string; use_tls: boolean; status: RouterStatus; remarks: string }
export interface NetworkRouterHealth { id: number; router_id: number; observed_at: string; connectivity_status: string; api_status: string; latency_ms: number; cpu_load: number; total_memory: number; free_memory: number; router_uptime: string; tcp_error: string; api_error: string }

export async function getNetworkRouters(): Promise<NetworkRouter[]> { return (await apiClient.get<{ routers: NetworkRouter[] }>('/network/routers')).data.routers }
export async function createNetworkRouter(data: NetworkRouterInput): Promise<NetworkRouter> { return (await apiClient.post<NetworkRouter>('/network/routers', data)).data }
export async function updateNetworkRouter(id: number, data: NetworkRouterInput): Promise<NetworkRouter> { return (await apiClient.put<NetworkRouter>(`/network/routers/${id}`, data)).data }
export async function testNetworkRouterConnection(id: number): Promise<NetworkRouter> { return (await apiClient.post<NetworkRouter>(`/network/routers/${id}/test-connection`)).data }
export async function setNetworkRouterCredentials(id: number, password: string): Promise<NetworkRouter> { return (await apiClient.put<NetworkRouter>(`/network/routers/${id}/credentials`, { password })).data }
export async function syncNetworkRouterResource(id: number): Promise<NetworkRouter> { return (await apiClient.post<NetworkRouter>(`/network/routers/${id}/sync-resource`)).data }
export async function getNetworkRouterHistory(id: number, limit = 100): Promise<NetworkRouterHealth[]> { return (await apiClient.get<{ history: NetworkRouterHealth[] }>(`/network/routers/${id}/history`, { params: { limit } })).data.history }
