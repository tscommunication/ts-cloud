import apiClient from './client'

export interface Customer {
  id: number
  customer_code: string
  full_name: string
  father_name: string
  mother_name: string
  mobile: string
  alt_mobile: string
  email: string
  nid: string
  division: string
  district: string
  upazila: string
  union: string
  village: string
  address: string
  status: string
  billing_day: number
}

export interface CustomersResponse {
  count: number
  customers: Customer[]
  page: number
  page_size: number
}

export interface CustomerListParams {
  search?: string
  status?: 'ACTIVE' | 'INACTIVE' | 'ARCHIVED' | ''
  page?: number
  page_size?: number
}

export interface CustomerSummary {
  subscriptions: number
  active_subscriptions: number
  invoices: number
  outstanding_amount: number
  successful_payments: number
  total_paid: number
}

export interface CreateCustomerRequest {
  full_name: string
  mobile: string
  father_name?: string
  mother_name?: string
  alt_mobile?: string
  email?: string
  nid?: string
  division?: string
  district?: string
  upazila?: string
  union?: string
  village?: string
  address?: string
  billing_day?: number
}

export type UpdateCustomerRequest = CreateCustomerRequest

export async function getCustomers(
  params: CustomerListParams = {},
): Promise<CustomersResponse> {
  const response = await apiClient.get<CustomersResponse>('/customers', {
    params,
  })
  return response.data
}

export async function createCustomer(
  data: CreateCustomerRequest,
): Promise<Customer> {
  const response = await apiClient.post<Customer>('/customers', data)
  return response.data
}

export async function updateCustomer(
  id: number,
  data: UpdateCustomerRequest,
): Promise<Customer> {
  const response = await apiClient.put<Customer>(`/customers/${id}`, data)
  return response.data
}

export async function updateCustomerStatus(
  id: number,
  status: 'ACTIVE' | 'INACTIVE',
): Promise<Customer> {
  const response = await apiClient.patch<Customer>(`/customers/${id}/status`, {
    status,
  })
  return response.data
}

export async function getCustomerSummary(id: number): Promise<CustomerSummary> {
  const response = await apiClient.get<CustomerSummary>(`/customers/${id}/summary`)
  return response.data
}

export async function archiveCustomer(id: number): Promise<Customer> {
  const response = await apiClient.post<Customer>(`/customers/${id}/archive`)
  return response.data
}
