import apiClient from './client'

interface APIResponse<T> {
  success: boolean
  data: T
  message?: string
}

export interface FTPServer {
  id: number
  name: string
  driver: string
  host: string
  port: number
  username: string
  root_path: string
  passive_port_start: number
  passive_port_end: number
  max_connections: number
  status: string
  description: string
  created_at: string
  updated_at: string
}

export interface FTPServerRequest {
  name: string
  driver: string
  host: string
  port: number
  username: string
  password: string
  root_path: string
  passive_port_start: number
  passive_port_end: number
  max_connections: number
  status: string
  description: string
}

export interface FTPUser {
  id: number
  customer_id: number
  subscription_id: number
  ftp_server_id: number
  username: string
  home_directory: string
  storage_quota_gb: number
  upload_limit_mbps: number
  download_limit_mbps: number
  status: string
  last_login?: string
  last_ip: string
  total_upload_bytes: number
  total_download_bytes: number
  remarks: string
  created_at: string
  updated_at: string
}

export interface FTPUserRequest {
  subscription_id: number
  ftp_server_id: number
  username: string
  password: string
  home_directory: string
  storage_quota_gb: number
  upload_limit_mbps: number
  download_limit_mbps: number
  status: string
  remarks: string
}

export interface FTPUserStats {
  id: number
  username: string
  status: string
  used_storage_bytes: number
  quota_bytes: number
  quota_gb: number
  usage_percent: number
  last_login?: string
  last_ip: string
  total_upload_bytes: number
  total_download_bytes: number
}

export interface FTPLoginLog {
  id: number
  ftp_user_id: number
  username: string
  ip_address: string
  login_status: string
  login_time: string
  user_agent: string
  created_at: string
}

const unwrap = <T>(response: { data: APIResponse<T> }) => response.data.data

export async function getFTPServers(): Promise<FTPServer[]> {
  return unwrap(await apiClient.get<APIResponse<FTPServer[]>>('/ftp-servers'))
}
export async function createFTPServer(data: FTPServerRequest): Promise<FTPServer> {
  return unwrap(await apiClient.post<APIResponse<FTPServer>>('/ftp-servers', data))
}
export async function updateFTPServer(id: number, data: FTPServerRequest): Promise<FTPServer> {
  return unwrap(await apiClient.put<APIResponse<FTPServer>>(`/ftp-servers/${id}`, data))
}
export async function deleteFTPServer(id: number): Promise<void> {
  await apiClient.delete(`/ftp-servers/${id}`)
}
export async function getFTPUsers(): Promise<FTPUser[]> {
  return unwrap(await apiClient.get<APIResponse<FTPUser[]>>('/ftp-users'))
}
export async function createFTPUser(data: FTPUserRequest): Promise<FTPUser> {
  return unwrap(await apiClient.post<APIResponse<FTPUser>>('/ftp-users', data))
}
export async function updateFTPUser(id: number, data: FTPUserRequest): Promise<FTPUser> {
  return unwrap(await apiClient.put<APIResponse<FTPUser>>(`/ftp-users/${id}`, data))
}
export async function deleteFTPUser(id: number): Promise<void> {
  await apiClient.delete(`/ftp-users/${id}`)
}
export async function suspendFTPUser(id: number): Promise<void> {
  await apiClient.post(`/ftp-users/${id}/suspend`)
}
export async function enableFTPUser(id: number): Promise<void> {
  await apiClient.post(`/ftp-users/${id}/enable`)
}
export async function getFTPUserStats(id: number): Promise<FTPUserStats> {
  return unwrap(await apiClient.get<APIResponse<FTPUserStats>>(`/ftp-users/${id}/stats`))
}
export async function getFTPLoginHistory(id: number): Promise<FTPLoginLog[]> {
  return unwrap(await apiClient.get<APIResponse<FTPLoginLog[]>>(`/ftp-users/${id}/login-history`))
}
