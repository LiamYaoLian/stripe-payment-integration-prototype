import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { getMe, login as apiLogin, register as apiRegister } from '../api/client'
import { ApiError } from '../api/client'
import { clearUserToken, getUserToken, saveUserToken } from '../lib/auth'

export type User = {
  id: string
  email: string
}

type AuthContextValue = {
  user: User | null
  isLoading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signUp: (email: string, password: string) => Promise<void>
  signOut: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const token = getUserToken()
    if (!token) {
      setIsLoading(false)
      return
    }

    let cancelled = false
    getMe(token)
      .then((profile) => {
        if (!cancelled) {
          setUser(profile)
        }
      })
      .catch((error) => {
        if (!cancelled) {
          if (error instanceof ApiError && error.status === 401) {
            clearUserToken()
          }
          setUser(null)
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
    saveUserToken(session.token)
    setUser(session.user)
  }, [])

  const signUp = useCallback(async (email: string, password: string) => {
    const session = await apiRegister(email, password)
    saveUserToken(session.token)
    setUser(session.user)
  }, [])

  const signOut = useCallback(() => {
    clearUserToken()
    setUser(null)
  }, [])

  const value = useMemo(
    () => ({ user, isLoading, signIn, signUp, signOut }),
    [user, isLoading, signIn, signUp, signOut],
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
