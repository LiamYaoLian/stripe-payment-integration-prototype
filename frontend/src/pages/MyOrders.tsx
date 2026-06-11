import { FormEvent, useEffect, useState } from 'react'
import { createGuestSession, listMyOrders } from '../api/client'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { clearGuestToken, getGuestToken, saveGuestToken } from '../lib/auth'
import { getCheckoutContext } from '../lib/checkoutContext'
import { formatPrice } from '../lib/formatPrice'
import type { Order } from '../types/api'

export function MyOrders() {
  const [email, setEmail] = useState('')
  const [orderId, setOrderId] = useState('')
  const [accessToken, setAccessToken] = useState('')
  const [orders, setOrders] = useState<Order[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [signedIn, setSignedIn] = useState(Boolean(getGuestToken()))

  useEffect(() => {
    const context = getCheckoutContext()
    if (context) {
      setOrderId(context.orderId)
      setAccessToken(context.accessToken)
    }
    const token = getGuestToken()
    if (token) {
      void loadOrders(token)
    }
  }, [])

  async function loadOrders(token: string) {
    setLoading(true)
    setError(null)
    try {
      const nextOrders = await listMyOrders(token)
      setOrders(nextOrders)
      setSignedIn(true)
    } catch (loadError) {
      clearGuestToken()
      setSignedIn(false)
      setOrders([])
      setError(loadError instanceof Error ? loadError.message : 'Failed to load orders')
    } finally {
      setLoading(false)
    }
  }

  async function handleSignIn(event: FormEvent) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const session = await createGuestSession(email.trim(), orderId.trim(), accessToken.trim())
      saveGuestToken(session.token)
      await loadOrders(session.token)
    } catch (signInError) {
      setError(signInError instanceof Error ? signInError.message : 'Sign in failed')
      setLoading(false)
    }
  }

  function handleSignOut() {
    clearGuestToken()
    setSignedIn(false)
    setOrders([])
  }

  if (loading && orders.length === 0 && !error) {
    return <LoadingMessage message="Loading your orders..." />
  }

  return (
    <div>
      <h1>My Orders</h1>
      <p>
        Sign in with the email on your order plus a recent order access token from checkout.
      </p>

      {!signedIn ? (
        <form className="card" onSubmit={handleSignIn}>
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
            Order ID
            <input
              type="text"
              value={orderId}
              onChange={(e) => setOrderId(e.target.value)}
              required
            />
          </label>
          <label>
            Access token
            <input
              type="text"
              value={accessToken}
              onChange={(e) => setAccessToken(e.target.value)}
              required
            />
          </label>
          <button className="btn" type="submit" disabled={loading}>
            View orders
          </button>
        </form>
      ) : (
        <p>
          Signed in.{' '}
          <button type="button" className="btn secondary" onClick={handleSignOut}>
            Sign out
          </button>
        </p>
      )}

      {error && <ErrorMessage message={error} />}

      {signedIn && orders.length === 0 && !error && !loading && (
        <p>No orders found for this email yet.</p>
      )}

      {orders.map((order) => (
        <div key={order.id} className="card">
          <h2>{order.orderNumber}</h2>
          <p>
            Status: <span className="status">{order.status}</span>
          </p>
          <p>
            Total: <strong>{formatPrice(order.totalAmountCents, order.currency)}</strong>
          </p>
          {order.items.length > 0 && (
            <ul>
              {order.items.map((item) => (
                <li key={`${order.id}-${item.productName}`}>
                  {item.productName} × {item.quantity}
                </li>
              ))}
            </ul>
          )}
        </div>
      ))}

      <BackToCatalogLink />
    </div>
  )
}
