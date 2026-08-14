import apiClient from './client'

export interface Payment {
  id: number
  receipt_no: string
  invoice_id: number
  customer_id: number
  subscription_id: number
	collected_by_user_id?: number
	collected_by_agent_id?: number
	collector_username: string
  payment_date: string
  amount: number
  method: string
  transaction_id: string
  status: string
  reference: string
  remarks: string
}

export interface CreatePaymentRequest {
  invoice_id: number
  payment_date: string
  amount: number
  method: string
  transaction_id: string
  reference: string
  remarks: string
}

export async function getPayments(): Promise<Payment[]> {
  const response = await apiClient.get<Payment[]>('/payments')
  return response.data
}

export async function getPayment(id: number): Promise<Payment> {
  const response = await apiClient.get<Payment>(`/payments/${id}`)
  return response.data
}

export async function createPayment(
  data: CreatePaymentRequest,
): Promise<Payment> {
  const response = await apiClient.post<Payment>('/payments', data)
  return response.data
}

export async function updatePayment(
  id: number,
  data: CreatePaymentRequest,
): Promise<Payment> {
  const response = await apiClient.put<Payment>(`/payments/${id}`, data)
  return response.data
}

export async function voidPayment(id: number): Promise<void> {
  await apiClient.post(`/payments/${id}/void`)
}
