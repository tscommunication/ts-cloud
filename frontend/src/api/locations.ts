import apiClient from './client'

export interface Division {
  id: number
  name: string
}

export interface District {
  id: number
  division_id: number
  name: string
}

export interface Upazila {
  id: number
  district_id: number
  name: string
}

export async function getDivisions(): Promise<Division[]> {
  const response = await apiClient.get<Division[]>('/divisions')
  return response.data
}

export async function getDistricts(
  divisionID: number,
): Promise<District[]> {
  const response = await apiClient.get<District[]>(
    `/divisions/${divisionID}/districts`,
  )
  return response.data
}

export async function getUpazilas(
  districtID: number,
): Promise<Upazila[]> {
  const response = await apiClient.get<Upazila[]>(
    `/districts/${districtID}/upazilas`,
  )
  return response.data
}
