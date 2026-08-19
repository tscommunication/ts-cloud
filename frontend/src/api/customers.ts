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
  country: string
  division: string
  district: string
  upazila: string
  post_office: string
  postal_code: string
  road_or_area: string
  village_or_holding: string
  union: string
  village: string
  address: string
  status: string
  billing_day: number
  pop_id?: number
  agent_id?: number

  date_of_birth: string
  joining_date: string
  occupation: string
  company_name: string
  designation: string
  nid_birth_date: string
  nid_issue_date: string
  nid_address: string
  present_address: string
  permanent_address: string
  tin: string
  customer_note: string
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
	cancelled_invoices: number
	voided_payments: number
	voided_amount: number
}

export interface CustomerLedgerEntry {
  date: string
  type: 'INVOICE' | 'PAYMENT'
  reference: string
  description: string
  debit: number
  credit: number
}

export interface CreateCustomerRequest {
  full_name: string
  mobile: string
  father_name?: string
  mother_name?: string
  alt_mobile?: string
  email?: string
  nid?: string
  country?: string
  division?: string
  district?: string
  upazila?: string
  post_office?: string
  postal_code?: string
  road_or_area?: string
  village_or_holding?: string
  union?: string
  village?: string
  address?: string
  billing_day?: number
  pop_id?: number
  agent_id?: number

  date_of_birth?: string
  joining_date?: string
  occupation?: string
  company_name?: string
  designation?: string
  nid_birth_date?: string
  nid_issue_date?: string
  nid_address?: string
  present_address?: string
  permanent_address?: string
  tin?: string
  customer_note?: string
}

export type UpdateCustomerRequest = CreateCustomerRequest

export interface CustomerTechnicalProfile {
  id: number
  customer_id: number

  onu_mac: string
  olt_pon: string
  olt_slot: string
  olt_port: string
  onu_type: string
  onu_model: string
  onu_ip: string
  onu_serial: string
  onu_sn: string

  router_brand: string
  router_model: string
  router_ip: string

  cable_type: string
  cable_length: number

  media_converter_mac: string
  media_converter_ip: string

  switch_model: string
  switch_port: string
  switch_ip: string

  additional_note: string

  onu_password_configured: boolean
  router_password_configured: boolean
  media_converter_password_configured: boolean
  switch_password_configured: boolean
}

export interface UpdateCustomerTechnicalProfileRequest {
  onu_mac?: string
  olt_pon?: string
  olt_slot?: string
  olt_port?: string
  onu_type?: string
  onu_model?: string
  onu_ip?: string
  onu_password?: string
  onu_serial?: string
  onu_sn?: string

  router_brand?: string
  router_model?: string
  router_ip?: string
  router_password?: string

  cable_type?: string
  cable_length?: number

  media_converter_mac?: string
  media_converter_ip?: string
  media_converter_password?: string

  switch_model?: string
  switch_port?: string
  switch_ip?: string
  switch_password?: string

  additional_note?: string
}

export interface CustomerReference {
  id: number
  customer_id: number
  name: string
  mobile: string
  address: string
  relation: string
  note: string
  created_at: string
  updated_at: string
}

export interface CustomerReferenceRequest {
  name: string
  mobile?: string
  address?: string
  relation?: string
  note?: string
}

export interface CustomerReferencesResponse {
  references: CustomerReference[]
}

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

export async function getCustomerLedger(id: number): Promise<CustomerLedgerEntry[]> {
  return (await apiClient.get<CustomerLedgerEntry[]>(`/customers/${id}/ledger`)).data
}

export async function getCustomerTechnicalProfile(
  id: number,
): Promise<CustomerTechnicalProfile | null> {
  const response = await apiClient.get<CustomerTechnicalProfile | null>(
    `/customers/${id}/technical-profile`,
  )
  return response.data
}

export async function updateCustomerTechnicalProfile(
  id: number,
  data: UpdateCustomerTechnicalProfileRequest,
): Promise<CustomerTechnicalProfile> {
  const response = await apiClient.put<CustomerTechnicalProfile>(
    `/customers/${id}/technical-profile`,
    data,
  )
  return response.data
}

export async function getCustomerReferences(
  id: number,
): Promise<CustomerReference[]> {
  const response = await apiClient.get<CustomerReferencesResponse>(
    `/customers/${id}/references`,
  )
  return response.data.references
}

export async function createCustomerReference(
  id: number,
  data: CustomerReferenceRequest,
): Promise<CustomerReference> {
  const response = await apiClient.post<CustomerReference>(
    `/customers/${id}/references`,
    data,
  )
  return response.data
}

export async function updateCustomerReference(
  customerID: number,
  referenceID: number,
  data: CustomerReferenceRequest,
): Promise<CustomerReference> {
  const response = await apiClient.put<CustomerReference>(
    `/customers/${customerID}/references/${referenceID}`,
    data,
  )
  return response.data
}

export async function deleteCustomerReference(
  customerID: number,
  referenceID: number,
): Promise<void> {
  await apiClient.delete(
    `/customers/${customerID}/references/${referenceID}`,
  )
}

export async function archiveCustomer(id: number): Promise<Customer> {
  const response = await apiClient.post<Customer>(`/customers/${id}/archive`)
  return response.data
}
