import { useCallback, useMemo } from 'react'

export function useAuth() {
  const token = useMemo(() => localStorage.getItem('token'), [])

  const isAuthenticated = useMemo(() => !!token, [token])

  const login = useCallback((newToken: string) => {
    localStorage.setItem('token', newToken)
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('token')
  }, [])

  return { token, isAuthenticated, login, logout, loading: false }
}
