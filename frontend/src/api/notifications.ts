import apiClient from './client'

export interface AppNotification {
  id: number
  type: 'CUSTOMER_CREATED' | 'NETWORK_ALERT'
  severity: 'INFO' | 'WARNING' | 'CRITICAL'
  title: string
  message: string
  target_path: string
  active: boolean
  read: boolean
  created_at: string
}

export async function getNotifications(): Promise<{ notifications: AppNotification[]; unread_count: number }> {
  const data = (await apiClient.get<{ notifications: AppNotification[] | null; unread_count: number | null }>('/notifications', { params: { limit: 25 } })).data
  return {
    notifications: data.notifications ?? [],
    unread_count: data.unread_count ?? 0,
  }
}

export async function markNotificationRead(id: number): Promise<void> { await apiClient.post(`/notifications/${id}/read`) }
export async function markAllNotificationsRead(): Promise<void> { await apiClient.post('/notifications/read-all') }
