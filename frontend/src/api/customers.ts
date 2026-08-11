import apiClient from './client'

export interface Customer {
  id: number
  customer_code: string
  full_name: string
  mobile: string
  email: string
  status: string
  billing_day: number
}

export interface CustomersResponse {
  count: number
  customers: Customer[]
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

export async function getCustomers(): Promise<CustomersResponse> {
  const response = await apiClient.get<CustomersResponse>('/customers')
  return response.data
}

export async function createCustomer(
  data: CreateCustomerRequest,
): Promise<Customer> {
  const response = await apiClient.post<Customer>('/customers', data)
  return response.data
}
