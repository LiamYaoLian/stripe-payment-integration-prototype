import { useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { LAST_EMBEDDED_PRODUCT_ID_KEY } from '../constants/checkout'
import { useOrderPolling } from '../hooks/useOrderPolling'
import { clearIdempotencyKey } from '../lib/idempotency'

export function CheckoutComplete() {
  const [params] = useSearchParams()
  const sessionId = params.get('session_id')
  const { order, error, isLoading } = useOrderPolling(sessionId)

  useEffect(() => {
    const productId = sessionStorage.getItem(LAST_EMBEDDED_PRODUCT_ID_KEY)
    if (productId) {
      clearIdempotencyKey(productId)
    }
  }, [])

  if (error) {
    return <ErrorMessage message={error} />
  }
  if (isLoading || !order) {
    return <LoadingMessage message="Confirming payment..." />
  }

  return (
    <div>
      <h1>Checkout Complete</h1>
      <p>
        Order: <strong>{order.orderNumber}</strong>
      </p>
      <p>
        Status: <span className="status">{order.status}</span>
      </p>
      <BackToCatalogLink />
    </div>
  )
}
