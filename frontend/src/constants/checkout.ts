import type { OrderStatus } from '../types/api'

export const TERMINAL_ORDER_STATUSES: readonly OrderStatus[] = [
  'paid',
  'failed',
  'expired',
  'canceled',
  'refunded',
]

export const POLL_MAX_ATTEMPTS = 30
export const POLL_INTERVAL_MS = 1000
export const POLL_TIMEOUT_MESSAGE =
  'Payment is still confirming. If you were charged, check your email for a receipt — your order may update shortly.'

export const CHECKOUT_IN_PROGRESS_MAX_RETRIES = 5
export const CHECKOUT_IN_PROGRESS_RETRY_MS = 500

export const LAST_EMBEDDED_PRODUCT_ID_KEY = 'last-embedded-product-id'
