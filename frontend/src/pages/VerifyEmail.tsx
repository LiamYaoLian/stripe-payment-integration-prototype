import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { verifyEmail } from '../api/client'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { useAuth } from '../context/AuthContext'

export function VerifyEmail() {
  const { refreshUser } = useAuth()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(Boolean(token))

  useEffect(() => {
    if (!token) {
      return
    }
    let cancelled = false
    verifyEmail(token)
      .then(async (result) => {
        if (!cancelled) {
          setMessage(result.message)
          await refreshUser()
        }
      })
      .catch((verifyError) => {
        if (!cancelled) {
          setError(verifyError instanceof Error ? verifyError.message : 'Verification failed')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [token, refreshUser])

  if (loading) {
    return <LoadingMessage message="Verifying your email..." />
  }

  return (
    <div>
      <h1>Verify email</h1>
      {message && <p>{message}</p>}
      {error && <ErrorMessage message={error} />}
      {!token && <p>Verification token is missing.</p>}
      <p>
        <Link to="/orders">Go to my orders</Link>
      </p>
      <BackToCatalogLink />
    </div>
  )
}
