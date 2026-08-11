
import apiClient from './client'

export interface Package {
  id: number
  package_code: string
  name: string
  price: number
  download_speed: number
  upload_speed: number
  burst_download: number
  burst_upload: number
  validity_days: number
  mikrotik_profile: string
  radius_profile: string
  status: string
  description: string
}

export interface CreatePackageRequest {
  name: string
  price: number
  download_speed: number
  upload_speed: number
  burst_download: number
  burst_upload: number
  validity_days: number
  mikrotik_profile: string
  radius_profile: string
  description: string
}

interface PackagesResponse {
  count: number
  packages: Package[]
}

export async function getPackages(): Promise<PackagesResponse> {
  const response = await apiClient.get<PackagesResponse>('/packages')
  return response.data
}

export async function getPackage(id: number): Promise<Package> {
  const response = await apiClient.get<Package>(`/packages/${id}`)
  return response.data
}

export async function createPackage(
  data: CreatePackageRequest,
): Promise<Package> {
  const response = await apiClient.post<Package>('/packages', data)
  return response.data
}

export async function updatePackage(
  id: number,
  data: CreatePackageRequest,
): Promise<Package> {
  const response = await apiClient.put<Package>(`/packages/${id}`, data)
  return response.data
}

export async function deletePackage(id: number): Promise<void> {
  await apiClient.delete(`/packages/${id}`)
}
