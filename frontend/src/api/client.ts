import {
  CHECKOUT_IN_PROGRESS_MAX_RETRIES,
  CHECKOUT_IN_PROGRESS_RETRY_MS,
} from '../constants/checkout'
import type { ApiEnvelope, CheckoutSessionResult, Order, Product } from '../types/api'

const API_BASE = import.meta.env.VITE_API_URL || ''

export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    Object.setPrototypeOf(this, ApiError.prototype)
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, init)
  } catch {
    throw new Error(
      API_BASE
        ? `Cannot reach API at ${API_BASE} — is the backend running on :8080?`
        : 'Cannot reach API — is the backend running? (make dev)',
    )
  }

  const body = (await response.json()) as ApiEnvelope<T>
  if (body.error) {
    throw new ApiError(response.status, body.error.code, body.error.message)
  }
  if (!response.ok) {
    throw new ApiError(response.status, 'UNKNOWN', 'request failed')
  }
  return body.data as T
}

export async function listProducts(): Promise<Product[]> {
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
  for (let attempt = 0; attempt < CHECKOUT_IN_PROGRESS_MAX_RETRIES; attempt++) {
    try {
      return await request<CheckoutSessionResult>('/api/checkout/sessions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
        body: JSON.stringify(body),
      })
    } catch (error) {
      if (
        error instanceof ApiError &&
        error.code === 'CHECKOUT_IN_PROGRESS' &&
        attempt < CHECKOUT_IN_PROGRESS_MAX_RETRIES - 1
      ) {
        await new Promise((resolve) => setTimeout(resolve, CHECKOUT_IN_PROGRESS_RETRY_MS))
        continue
      }
      throw error
    }
  }
  throw new Error('checkout failed')
}

export async function getOrderBySession(sessionId: string): Promise<Order> {
  return request<Order>(`/api/orders/by-session/${sessionId}`)
}
