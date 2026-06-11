import { useEffect, useState } from 'react'
import { listMyOrders } from '../api/client'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { useAuth } from '../context/AuthContext'
import { formatPrice } from '../lib/formatPrice'
import type { Order } from '../types/api'

export function MyOrders() {
  const { user, signOut } = useAuth()
  const [orders, setOrders] = useState<Order[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    listMyOrders()
      .then((nextOrders) => {
        if (!cancelled) {
          setOrders(nextOrders)
        }
      })
      .catch((loadError) => {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : 'Failed to load orders')
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
  }, [])

  if (loading) {
    return <LoadingMessage message="Loading your orders..." />
  }

  return (
    <div>
      <h1>My Orders</h1>
      <p>
        Signed in as <strong>{user?.email}</strong>.{' '}
        <button type="button" className="btn secondary" onClick={() => void signOut()}>
          Sign out
        </button>
      </p>

      {error && <ErrorMessage message={error} />}

      {!error && orders.length === 0 && (
        <p>No orders yet — checkout from the catalog.</p>
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
