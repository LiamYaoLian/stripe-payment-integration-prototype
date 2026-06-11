export type OrderStatus =
  | 'pending'
  | 'processing'
  | 'paid'
  | 'failed'
  | 'expired'
  | 'canceled'

export interface Product {
  id: string
  name: string
  description?: string
  unitAmountCents: number
  currency: string
}

export interface OrderItem {
  productName: string
  quantity: number
  lineTotalCents: number
}

export interface Order {
  id: string
  orderNumber: string
  status: OrderStatus
  totalAmountCents: number
  currency: string
  paidAt?: string
  items: OrderItem[]
}

export interface CheckoutSessionResult {
  orderId: string
  orderNumber: string
  sessionId: string
  url?: string
  clientSecret?: string
}

export interface ApiEnvelope<T> {
  data: T | null
  error: { code: string; message: string } | null
}
