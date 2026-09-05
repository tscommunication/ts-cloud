import apiClient from "./client";

export interface CurrentUser {
  id: number;
  username: string;
  role: string;
}

export interface User {
  id: number;
  name: string;
  username: string;
  email: string;
  role: string;
  active: boolean;
  agent_id?: number;
}

export interface UpdateUserRequest {
  name?: string;
  username?: string;
  email?: string;
  password?: string;
  role?: string;
  active?: boolean;
  agent_id?: number;
}

export interface CreateUserRequest {
  name: string;
  username: string;
  email: string;
  password: string;
  role: "superadmin" | "admin" | "noc" | "agent" | "user";
  agent_id?: number;
}

interface UsersResponse {
  page: number;
  limit: number;
  total: number;
  count: number;
  users: User[];
}

export async function getCurrentUser(): Promise<CurrentUser> {
  const response = await apiClient.get<CurrentUser>("/me");
  return response.data;
}

export async function getUser(id: number): Promise<User> {
  const response = await apiClient.get<User>(`/users/${id}`);
  return response.data;
}

export async function updateUser(
  id: number,
  data: UpdateUserRequest,
): Promise<User> {
  const response = await apiClient.put<User>(`/users/${id}`, data);
  return response.data;
}

export async function getUsers(): Promise<UsersResponse> {
  const response = await apiClient.get<UsersResponse>("/users", {
    params: { page: 1, limit: 100, sort: "id", order: "desc" },
  });
  return response.data;
}

export async function createUser(data: CreateUserRequest): Promise<User> {
  const response = await apiClient.post<User>("/users", data);
  return response.data;
}

export async function deleteUser(id: number): Promise<void> {
  await apiClient.delete(`/users/${id}`);
}

export async function changeMyPassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  await apiClient.post("/me/password", {
    current_password: currentPassword,
    new_password: newPassword,
  });
}
