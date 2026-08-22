import apiClient from './client'

export type ProvisionRequestStatus =
  | 'PENDING'
  | 'APPROVED'
  | 'REJECTED'
  | 'CANCELLED'
  | 'COMPLETED'

export interface ProvisionPackage {
  id: number
  package_code: string
  name: string
  price: number
  download_speed: number
  upload_speed: number
  status: string
}

export interface ProvisionRouter {
  id: number
  code: string
  name: string
  pop_id?: number
  pop_name: string
  status: string
}

export interface CustomerProvisionRequest {
  id: number
  request_code: string
  source: string
  status: ProvisionRequestStatus

  agent_id?: number
  pop_id?: number

  full_name: string
  mobile: string
  father_name: string
  mother_name: string
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
  latitude?: number | null
  longitude?: number | null

  package_id: number
  router_id: number

  pppoe_username: string

  billing_day: number
  activation_date: string

  remarks: string

  requested_by_user_id: number
  requested_at: string

  reviewed_by_user_id?: number
  reviewed_at?: string

  rejection_reason: string

  customer_id?: number
  subscription_id?: number

  created_at: string
}

export interface CreateCustomerProvisionRequestInput {
  full_name: string
  mobile: string
  father_name?: string
  mother_name?: string
  alt_mobile?: string
  email?: string
  nid: string

  country?: string
  division?: string
  district?: string
  upazila?: string
  post_office?: string
  postal_code?: string
  road_or_area?: string
  village_or_holding?: string
  latitude?: number | null
  longitude?: number | null

  package_id: number
  router_id?: number

  pppoe_username: string
  pppoe_password?: string

  billing_day: number
  activation_date?: string

  remarks?: string
}

interface ProvisionRequestsResponse {
  count: number
  requests: CustomerProvisionRequest[]
}

interface ProvisionPackagesResponse {
  count: number
  packages: ProvisionPackage[]
}

interface ProvisionRoutersResponse {
  count: number
  routers: ProvisionRouter[]
}

export async function getProvisionPackages(): Promise<ProvisionPackage[]> {
  const response =
    await apiClient.get<ProvisionPackagesResponse>(
      '/provision-catalog/packages',
    )

  return response.data.packages
}

export async function getProvisionRouters(): Promise<ProvisionRouter[]> {
  const response =
    await apiClient.get<ProvisionRoutersResponse>(
      '/provision-catalog/routers',
    )

  return response.data.routers
}

export async function createProvisionRequest(
  data: CreateCustomerProvisionRequestInput,
): Promise<CustomerProvisionRequest> {
  const response =
    await apiClient.post<CustomerProvisionRequest>(
      '/customer-provision-requests',
      data,
    )

  return response.data
}

export async function getProvisionRequests(
  status: '' | ProvisionRequestStatus = '',
): Promise<ProvisionRequestsResponse> {
  const response =
    await apiClient.get<ProvisionRequestsResponse>(
      '/customer-provision-requests',
      {
        params: status ? { status } : undefined,
      },
    )

  return response.data
}

export async function getProvisionRequest(
  id: number,
): Promise<CustomerProvisionRequest> {
  const response =
    await apiClient.get<CustomerProvisionRequest>(
      `/customer-provision-requests/${id}`,
    )

  return response.data
}

export async function approveProvisionRequest(
  id: number,
): Promise<CustomerProvisionRequest> {
  const response =
    await apiClient.post<CustomerProvisionRequest>(
      `/customer-provision-requests/${id}/approve`,
    )

  return response.data
}

export async function rejectProvisionRequest(
  id: number,
  reason: string,
): Promise<CustomerProvisionRequest> {
  const response =
    await apiClient.post<CustomerProvisionRequest>(
      `/customer-provision-requests/${id}/reject`,
      { reason },
    )

  return response.data
}
