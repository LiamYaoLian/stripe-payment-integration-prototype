import { useSearchParams } from 'react-router-dom'
import { BackToCatalogLink } from '../components/BackToCatalogLink'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { useOrderPolling } from '../hooks/useOrderPolling'

export function CheckoutSuccess() {
  const [params] = useSearchParams()
  const sessionId = params.get('session_id')
  const { order, error, isLoading } = useOrderPolling(sessionId)

  if (error) {
    return <ErrorMessage message={error} />
  }
  if (isLoading || !order) {
    return <LoadingMessage message="Confirming payment..." />
  }

  return (
    <div>
      <h1>Payment Status</h1>
      <p>
        Order: <strong>{order.orderNumber}</strong>
      </p>
      <p>
        Status: <span className="status">{order.status}</span>
      </p>
      {order.status === 'paid' && <p>Thank you! Your payment was successful.</p>}
      {(order.status === 'pending' || order.status === 'processing') && (
        <p>Waiting for payment confirmation...</p>
      )}
      <BackToCatalogLink />
    </div>
  )
}
