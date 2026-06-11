import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'

export function SignUp() {
  const { signUp, user } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  if (user) {
    navigate('/orders', { replace: true })
    return null
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (password !== confirmPassword) {
      setError('Passwords do not match')
      return
    }
    setLoading(true)
    setError(null)
    try {
      await signUp(email.trim(), password)
      navigate('/orders', { replace: true })
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Sign up failed')
      setLoading(false)
    }
  }

  return (
    <div>
      <h1>Sign up</h1>
      <form className="card" onSubmit={handleSubmit}>
        <label>
          Email
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
        </label>
        <label>
          Password
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
        <button className="btn" type="submit" disabled={loading}>
          {loading ? 'Creating account...' : 'Sign up'}
        </button>
      </form>
      {error && <ErrorMessage message={error} />}
      <p>
        Already have an account? <Link to="/login">Sign in</Link>
      </p>
      <BackToCatalogLink />
    </div>
  )
}
