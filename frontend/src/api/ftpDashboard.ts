import apiClient from './client'

export interface FTPDashboardData {
  total_users: number
  online_users: number
  today_logins: number
  today_uploads: number
  today_downloads: number
  today_upload_bytes: number
  today_download_bytes: number
}

export interface FTPDashboardResponse {
  data: FTPDashboardData
  success: boolean
}

export async function getFTPDashboard(): Promise<FTPDashboardData> {
  const response = await apiClient.get<FTPDashboardResponse>(
    '/ftp-dashboard',
  )

  return response.data.data
}
