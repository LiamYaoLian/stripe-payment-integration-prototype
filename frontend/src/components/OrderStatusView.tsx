import type { Order } from '../types/api'
import { BackToCatalogLink } from './BackToCatalogLink'

interface OrderStatusViewProps {
  title: string
  order: Order
  showPaidMessage?: boolean
  showPendingMessage?: boolean
}

export function OrderStatusView({
  title,
  order,
  showPaidMessage = false,
  showPendingMessage = false,
}: OrderStatusViewProps) {
  return (
    <div>
      <h1>{title}</h1>
      <p>
        Order: <strong>{order.orderNumber}</strong>
      </p>
      <p>
        Status: <span className="status">{order.status}</span>
      </p>
      {showPaidMessage && order.status === 'paid' && (
        <p>Thank you! Your payment was successful.</p>
      )}
      {showPendingMessage &&
        (order.status === 'pending' || order.status === 'processing') && (
          <p>Waiting for payment confirmation...</p>
        )}
      <BackToCatalogLink />
    </div>
  )
}
