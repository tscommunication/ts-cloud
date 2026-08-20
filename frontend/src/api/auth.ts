import apiClient from './client'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginUser {
  id: number
  username: string
  role: string
  agent_id?: number
  customer_id?: number
}

export interface LoginResponse {
  message: string
  access_token: string
  user: LoginUser
}

export async function login(
  credentials: LoginRequest,
): Promise<LoginResponse> {
  const response = await apiClient.post<LoginResponse>(
    '/auth/login',
    credentials,
  )

  return response.data
}

export function logout(): void {
  localStorage.removeItem('access_token')
  localStorage.removeItem('user')
}

export function getStoredUser(): LoginUser | null {
  const user = localStorage.getItem('user')

  if (!user) {
    return null
  }

  try {
    return JSON.parse(user) as LoginUser
  } catch {
    return null
  }
}

export function isAuthenticated(): boolean {
  return Boolean(localStorage.getItem('access_token'))
}
