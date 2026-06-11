const API_BASE = import.meta.env.VITE_API_URL || ''

type ApiEnvelope<T> = {
  data: T | null
  error: { code: string; message: string } | null
}

export class ApiError extends Error {
  code: string
  status: number

  constructor(status: number, code: string, message: string) {
    super(message)
    this.code = code
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response
  try {
    res = await fetch(`${API_BASE}${path}`, init)
  } catch {
    throw new Error(
      API_BASE
        ? `Cannot reach API at ${API_BASE} — is the backend running on :8080?`
        : 'Cannot reach API — is the backend running? (make dev)',
    )
  }
  const body = (await res.json()) as ApiEnvelope<T>
  if (body.error) {
    throw new ApiError(res.status, body.error.code, body.error.message)
  }
  if (!res.ok) {
    throw new ApiError(res.status, 'UNKNOWN', 'request failed')
  }
  return body.data as T
}

export type Product = {
  id: string
  name: string
  description?: string
  unitAmountCents: number
  currency: string
}

export type Order = {
  id: string
  orderNumber: string
  status: string
  totalAmountCents: number
  currency: string
  paidAt?: string
  items: { productName: string; quantity: number; lineTotalCents: number }[]
}

export type CheckoutSessionResult = {
  orderId: string
  orderNumber: string
  sessionId: string
  url?: string
  clientSecret?: string
}

export function getIdempotencyKey(productId: string): string {
  const storageKey = `checkout-idempotency-${productId}`
  let key = sessionStorage.getItem(storageKey)
  if (!key) {
    key = crypto.randomUUID()
    sessionStorage.setItem(storageKey, key)
  }
  return key
}

export function clearIdempotencyKey(productId: string) {
  sessionStorage.removeItem(`checkout-idempotency-${productId}`)
}

export async function listProducts() {
  const data = await request<{ products: Product[] }>('/api/products')
  return data.products
}

export async function createCheckoutSession(
  body: {
    uiMode: 'hosted' | 'embedded'
    items: { productId: string; quantity: number }[]
    customerEmail?: string
  },
  idempotencyKey: string,
): Promise<CheckoutSessionResult> {
  const maxRetries = 5
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await request<CheckoutSessionResult>('/api/checkout/sessions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
        body: JSON.stringify(body),
      })
    } catch (err) {
      if (err instanceof ApiError && err.code === 'CHECKOUT_IN_PROGRESS' && i < maxRetries - 1) {
        await new Promise((r) => setTimeout(r, 500))
        continue
      }
      throw err
    }
  }
  throw new Error('checkout failed')
}

export async function getOrderBySession(sessionId: string) {
  return request<Order>(`/api/orders/by-session/${sessionId}`)
}
