import { FormEvent, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { resetPassword } from '../api/client'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'

export function ResetPassword() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (password !== confirmPassword) {
      setError('Passwords do not match')
      return
    }
    if (!token) {
      setError('Reset token is missing')
      return
    }
    setLoading(true)
    setError(null)
    try {
      const result = await resetPassword(token, password)
      setMessage(result.message)
      navigate('/login', { replace: true })
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Reset failed')
      setLoading(false)
    }
  }

  return (
    <div>
      <h1>Reset password</h1>
      <form className="card" onSubmit={handleSubmit}>
        <label>
          New password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="new-password"
            minLength={8}
          />
        </label>
        <label>
          Confirm password
          <input
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
            autoComplete="new-password"
            minLength={8}
          />
        </label>
        <button className="btn" type="submit" disabled={loading || !token}>
          {loading ? 'Updating...' : 'Update password'}
        </button>
      </form>
      {message && <p>{message}</p>}
      {error && <ErrorMessage message={error} />}
      {!token && <p>Invalid reset link. Request a new one from the forgot password page.</p>}
      <p>
        <Link to="/forgot-password">Request a new reset link</Link>
      </p>
      <BackToCatalogLink />
    </div>
  )
}
