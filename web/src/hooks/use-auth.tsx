import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import {
  type AuthState,
  type LoginRequest,
  login as apiLogin,
  signup as apiSignup,
  refreshToken as apiRefreshToken,
  getStoredAuth,
  setStoredAuth,
  clearStoredAuth,
} from '@/api/auth'

interface AuthContextValue {
  isAuthenticated: boolean
  isLoading: boolean
  user: AuthState['user']
  token: string | null
  login: (credentials: LoginRequest) => Promise<void>
  signup: (credentials: LoginRequest) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [authState, setAuthState] = useState<AuthState | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const init = async () => {
      try {
        const stored = getStoredAuth()
        if (stored?.token && stored?.expiresAt) {
          const expiresAt = new Date(stored.expiresAt)
          const now = new Date()
          const fiveMinutes = 5 * 60 * 1000

          if (expiresAt.getTime() - now.getTime() < fiveMinutes && stored.refreshToken) {
            try {
              const refreshed = await apiRefreshToken(stored.refreshToken)
              const newState = {
                ...stored,
                token: refreshed.token,
                expiresAt: refreshed.expiresAt,
              }
              setStoredAuth(newState)
              setAuthState(newState)
            } catch {
              clearStoredAuth()
            }
          } else if (expiresAt > now) {
            setAuthState(stored)
          } else {
            clearStoredAuth()
          }
        }
      } finally {
        setIsLoading(false)
      }
    }

    init()
  }, [])

  useEffect(() => {
    if (!authState?.token || !authState?.expiresAt || !authState?.refreshToken) return

    const expiresAt = new Date(authState.expiresAt)
    const now = new Date()
    const timeUntilExpiry = expiresAt.getTime() - now.getTime()
    const refreshTime = timeUntilExpiry - 60000

    if (refreshTime <= 0) return

    const timeout = setTimeout(async () => {
      try {
        const refreshed = await apiRefreshToken(authState.refreshToken!)
        const newState = {
          ...authState,
          token: refreshed.token,
          expiresAt: refreshed.expiresAt,
        }
        setStoredAuth(newState)
        setAuthState(newState)
      } catch {
        clearStoredAuth()
        setAuthState(null)
      }
    }, refreshTime)

    return () => clearTimeout(timeout)
  }, [authState])

  const login = useCallback(async (credentials: LoginRequest) => {
    const response = await apiLogin(credentials)
    const state: AuthState = {
      token: response.token,
      refreshToken: response.refreshToken,
      expiresAt: response.expiresAt,
      user: response.user,
    }
    setStoredAuth(state)
    setAuthState(state)
  }, [])

  const signup = useCallback(async (credentials: LoginRequest) => {
    const response = await apiSignup(credentials)
    const state: AuthState = {
      token: response.token,
      refreshToken: response.refreshToken,
      expiresAt: response.expiresAt,
      user: response.user,
    }
    setStoredAuth(state)
    setAuthState(state)
  }, [])

  const logout = useCallback(() => {
    clearStoredAuth()
    setAuthState(null)
  }, [])

  const value: AuthContextValue = {
    isAuthenticated: !!authState?.token,
    isLoading,
    user: authState?.user ?? null,
    token: authState?.token ?? null,
    login,
    signup,
    logout,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
