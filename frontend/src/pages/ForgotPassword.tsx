import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { forgotPassword } from '../api/client'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'

export function ForgotPassword() {
  const [email, setEmail] = useState('')
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    setMessage(null)
    try {
      const result = await forgotPassword(email.trim())
      setMessage(result.message)
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Request failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <h1>Forgot password</h1>
      <p>Enter your account email and we will send a reset link if the account exists.</p>
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
        <button className="btn" type="submit" disabled={loading}>
          {loading ? 'Sending...' : 'Send reset link'}
        </button>
      </form>
      {message && <p>{message}</p>}
      {error && <ErrorMessage message={error} />}
      <p>
        <Link to="/login">Back to sign in</Link>
      </p>
      <BackToCatalogLink />
    </div>
  )
}
