import type { Order } from '../../types/api'

export const paidOrder: Order = {
  id: 'ord1',
  orderNumber: 'ORD-1',
  status: 'paid',
  totalAmountCents: 4900,
  currency: 'usd',
  items: [{ productName: 'Pro', quantity: 1, lineTotalCents: 4900 }],
}
