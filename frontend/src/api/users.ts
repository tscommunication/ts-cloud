import apiClient from './client'

export interface CurrentUser {
  id: number
  username: string
  role: string
}

export interface User {
  id: number
  name: string
  username: string
  email: string
  role: string
  active: boolean
}

export interface UpdateUserRequest {
  name?: string
  username?: string
  email?: string
  password?: string
}

export async function getCurrentUser(): Promise<CurrentUser> {
  const response = await apiClient.get<CurrentUser>('/me')
  return response.data
}

export async function getUser(id: number): Promise<User> {
  const response = await apiClient.get<User>(`/users/${id}`)
  return response.data
}

export async function updateUser(
  id: number,
  data: UpdateUserRequest,
): Promise<User> {
  const response = await apiClient.put<User>(`/users/${id}`, data)
  return response.data
}
