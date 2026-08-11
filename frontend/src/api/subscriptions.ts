import apiClient from './client'

export interface Subscription {
  id: number
  subscription_code: string
  customer_id: number
  package_id: number
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
  status: string
  remarks: string
}

interface SubscriptionsResponse {
  count: number
  subscriptions: Subscription[]
}

export async function getSubscriptions(): Promise<SubscriptionsResponse> {
  const response =
    await apiClient.get<SubscriptionsResponse>(
      '/subscriptions',
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

export async function deleteSubscription(
  id: number,
): Promise<void> {
  await apiClient.delete(`/subscriptions/${id}`)
}
