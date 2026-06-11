import { useSearchParams } from 'react-router-dom'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { OrderStatusView } from '../components/OrderStatusView'
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
    <OrderStatusView
      title="Payment Status"
      order={order}
      showPaidMessage
      showPendingMessage
    />
  )
}
