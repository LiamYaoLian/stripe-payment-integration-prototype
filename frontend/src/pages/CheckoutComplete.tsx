import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { clearIdempotencyKey, getOrderBySession, Order } from '../api/client'

export default function CheckoutComplete() {
  const [params] = useSearchParams()
  const sessionId = params.get('session_id')
  const [order, setOrder] = useState<Order | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const productId = sessionStorage.getItem('last-embedded-product-id')
    if (productId) clearIdempotencyKey(productId)

    if (!sessionId) {
      setError('Missing session_id')
      return
    }

    let attempts = 0
    const poll = async () => {
      try {
        const o = await getOrderBySession(sessionId)
        setOrder(o)
        if (['paid', 'failed', 'expired', 'canceled'].includes(o.status)) return
        if (attempts++ < 30) setTimeout(poll, 1000)
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
      <h1>Checkout Complete</h1>
      <p>Order: <strong>{order.orderNumber}</strong></p>
      <p>Status: <span className="status">{order.status}</span></p>
      <Link className="btn" to="/">Back to catalog</Link>
    </div>
  )
}
