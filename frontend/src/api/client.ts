import {
  CHECKOUT_IN_PROGRESS_MAX_RETRIES,
  CHECKOUT_IN_PROGRESS_RETRY_MS,
} from '../constants/checkout'
import type { CheckoutSessionResult, Order, Product } from '../types/api'
import {
  ApiResponseError,
  authSessionSchema,
  checkoutSessionResultSchema,
  orderSchema,
  parseApiEnvelope,
  productSchema,
  userProfileSchema,
} from './schemas'
import { getUserToken } from '../lib/auth'
import { z } from 'zod'

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

async function request<T>(path: string, dataSchema: z.ZodType<T>, init?: RequestInit): Promise<T> {
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

  let body: unknown
  try {
    body = await response.json()
  } catch {
    throw new ApiError(response.status, 'INVALID_JSON', 'response is not valid JSON')
  }

  try {
    const data = parseApiEnvelope(dataSchema, body)
    if (!response.ok) {
      throw new ApiError(response.status, 'UNKNOWN', 'request failed')
    }
    return data
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    if (error instanceof z.ZodError) {
      throw new ApiError(response.status, 'INVALID_RESPONSE', 'API response failed validation')
    }
    if (error instanceof ApiResponseError) {
      throw new ApiError(response.status, error.code, error.message)
    }
    throw error
  }
}

export async function listProducts(): Promise<Product[]> {
  const data = await request('/api/products', z.object({ products: z.array(productSchema) }))
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
      return await request('/api/checkout/sessions', checkoutSessionResultSchema, {
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

export async function register(
  email: string,
  password: string,
): Promise<{ token: string; expiresAt: string; user: { id: string; email: string } }> {
  return request('/api/auth/register', authSessionSchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
}

export async function login(
  email: string,
  password: string,
): Promise<{ token: string; expiresAt: string; user: { id: string; email: string } }> {
  return request('/api/auth/login', authSessionSchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
}

export async function getMe(token: string): Promise<{ id: string; email: string }> {
  return request('/api/auth/me', userProfileSchema, {
    headers: { Authorization: `Bearer ${token}` },
  })
}

export async function listMyOrders(token?: string): Promise<Order[]> {
  const authToken = token ?? getUserToken()
  if (!authToken) {
    throw new ApiError(401, 'UNAUTHORIZED', 'not signed in')
  }
  const data = await request('/api/orders/mine', z.object({ orders: z.array(orderSchema) }), {
    headers: { Authorization: `Bearer ${authToken}` },
  })
  return data.orders
}

export async function getOrderBySession(sessionId: string, accessToken?: string | null): Promise<Order> {
  const headers: Record<string, string> = {}
  if (accessToken) {
    headers['X-Order-Token'] = accessToken
  }
  return request(`/api/orders/by-session/${sessionId}`, orderSchema, { headers })
}
