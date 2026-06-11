import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { getOrderBySession, Order } from '../api/client'

export default function CheckoutSuccess() {
  const [params] = useSearchParams()
  const sessionId = params.get('session_id')
  const [order, setOrder] = useState<Order | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!sessionId) {
      setError('Missing session_id')
      return
    }

    let attempts = 0
    const maxAttempts = 30

    const poll = async () => {
      try {
        const o = await getOrderBySession(sessionId)
        setOrder(o)
        if (['paid', 'failed', 'expired', 'canceled'].includes(o.status)) return
        if (attempts++ < maxAttempts) {
          setTimeout(poll, 1000)
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load order')
      }
    }

    poll()
  }, [sessionId])

  if (error) return <p className="error">{error}</p>
  if (!order) return <p>Confirming payment...</p>

  return (
    <div>
      <h1>Payment Status</h1>
      <p>Order: <strong>{order.orderNumber}</strong></p>
      <p>Status: <span className="status">{order.status}</span></p>
      {order.status === 'paid' && <p>Thank you! Your payment was successful.</p>}
      {order.status === 'pending' || order.status === 'processing' ? (
        <p>Waiting for payment confirmation...</p>
      ) : null}
      <Link className="btn" to="/">Back to catalog</Link>
    </div>
  )
}
