import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { getMe, login as apiLogin, logout as apiLogout, register as apiRegister } from '../api/client'
import { ApiError } from '../api/client'

export type User = {
  id: string
  email: string
  emailVerified: boolean
}

type AuthContextValue = {
  user: User | null
  isLoading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signUp: (email: string, password: string) => Promise<void>
  signOut: () => Promise<void>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  const refreshUser = useCallback(async () => {
    try {
      const profile = await getMe()
      setUser(profile)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setUser(null)
        return
      }
      throw error
    }
  }, [])

  useEffect(() => {
    let cancelled = false

    getMe()
      .then((profile) => {
        if (!cancelled) {
          setUser(profile)
        }
      })
      .catch((error) => {
        if (!cancelled) {
          if (!(error instanceof ApiError && error.status === 401)) {
            setUser(null)
          } else {
            setUser(null)
          }
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [])

  const signIn = useCallback(async (email: string, password: string) => {
    const session = await apiLogin(email, password)
    setUser(session.user)
  }, [])

  const signUp = useCallback(async (email: string, password: string) => {
    const session = await apiRegister(email, password)
    setUser(session.user)
  }, [])

  const signOut = useCallback(async () => {
    try {
      await apiLogout()
    } finally {
      setUser(null)
    }
  }, [])

  const value = useMemo(
    () => ({ user, isLoading, signIn, signUp, signOut, refreshUser }),
    [user, isLoading, signIn, signUp, signOut, refreshUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
