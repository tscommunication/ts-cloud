import apiClient from './client'

export interface Subscription {
  id: number
  subscription_code: string
  customer_id: number
  package_id: number
  customer_code: string
  customer_name: string
  package_code: string
  package_name: string
  status: string
  activation_date: string
  next_billing_date: string
  expiry_date: string
  billing_day: number
  router_id: number
  pppoe_username: string
  remarks: string
}

export interface CreateSubscriptionRequest {
  customer_id: number
  package_id: number
  activation_date: string
  billing_day: number
  router_id: number
  pppoe_username: string
  pppoe_password: string
  remarks: string
}

export interface UpdateSubscriptionRequest {
  billing_day: number
  router_id: number
  pppoe_username: string
  pppoe_password: string
  remarks: string
}

interface SubscriptionsResponse {
  count: number
  subscriptions: Subscription[]
}

export interface SubscriptionListParams {
  status?: 'ACTIVE' | 'SUSPENDED' | 'EXPIRED' | 'DISCONNECTED' | ''
  expiring_within_days?: number
}

export async function getSubscriptions(
  params: SubscriptionListParams = {},
): Promise<SubscriptionsResponse> {
  const response =
    await apiClient.get<SubscriptionsResponse>(
      '/subscriptions',
      { params },
    )

  return response.data
}

export async function getSubscription(
  id: number,
): Promise<Subscription> {
  const response = await apiClient.get<Subscription>(
    `/subscriptions/${id}`,
  )

  return response.data
}

export async function createSubscription(
  data: CreateSubscriptionRequest,
): Promise<Subscription> {
  const response = await apiClient.post<Subscription>(
    '/subscriptions',
    data,
  )

  return response.data
}

export async function updateSubscription(
  id: number,
  data: UpdateSubscriptionRequest,
): Promise<Subscription> {
  const response = await apiClient.put<Subscription>(
    `/subscriptions/${id}`,
    data,
  )

  return response.data
}

export async function disconnectSubscription(
  id: number,
): Promise<Subscription> {
  const response = await apiClient.post<Subscription>(`/subscriptions/${id}/disconnect`)
  return response.data
}

export async function suspendSubscription(id: number): Promise<Subscription> {
  const response = await apiClient.post<Subscription>(`/subscriptions/${id}/suspend`)
  return response.data
}

export async function activateSubscription(id: number): Promise<Subscription> {
  const response = await apiClient.post<Subscription>(`/subscriptions/${id}/activate`)
  return response.data
}

export async function renewSubscription(
  id: number,
  months: number,
): Promise<Subscription> {
  const response = await apiClient.post<Subscription>(`/subscriptions/${id}/renew`, {
    months,
  })
  return response.data
}
