import axios from 'axios'

const API_BASE = '/api/v1'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  refreshToken: string
  expiresAt: string
  user: {
    id: string
    username: string
    role: string
  }
}

export interface AuthState {
  token: string | null
  refreshToken: string | null
  user: {
    id: string
    username: string
    role: string
  } | null
  expiresAt: string | null
}

const AUTH_STORAGE_KEY = 'podling_auth'

export function getStoredAuth(): AuthState | null {
  const stored = localStorage.getItem(AUTH_STORAGE_KEY)
  if (!stored) return null
  try {
    return JSON.parse(stored)
  } catch {
    return null
  }
}

export function setStoredAuth(auth: AuthState): void {
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(auth))
}

export function clearStoredAuth(): void {
  localStorage.removeItem(AUTH_STORAGE_KEY)
}

export async function login(credentials: LoginRequest): Promise<LoginResponse> {
  const response = await axios.post<LoginResponse>(
    `${API_BASE}/auth/login`,
    credentials
  )
  return response.data
}

export async function signup(credentials: LoginRequest): Promise<LoginResponse> {
  const response = await axios.post<LoginResponse>(
    `${API_BASE}/auth/signup`,
    credentials
  )
  return response.data
}

export async function refreshToken(token: string): Promise<{ token: string; expiresAt: string }> {
  const response = await axios.post<{ token: string; expiresAt: string }>(
    `${API_BASE}/auth/refresh`,
    {refreshToken: token}
  )
  return response.data
}

export async function getCurrentUser(token: string): Promise<{
  userId: string
  username: string
  role: string
}> {
  const response = await axios.get(`${API_BASE}/auth/me`, {
    headers: {Authorization: `Bearer ${token}`},
  })
  return response.data
}
