import {
  CHECKOUT_IN_PROGRESS_MAX_RETRIES,
  CHECKOUT_IN_PROGRESS_RETRY_MS,
} from '../constants/checkout'
import type { CheckoutSessionResult, Order, Product } from '../types/api'
import {
  ApiResponseError,
  authSessionSchema,
  checkoutSessionResultSchema,
  messageSchema,
  orderSchema,
  parseApiEnvelope,
  productSchema,
  userProfileSchema,
} from './schemas'
import { z } from 'zod'

const API_BASE = import.meta.env.VITE_API_URL || ''

const credentials: RequestCredentials = 'include'

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
    response = await fetch(`${API_BASE}${path}`, { credentials, ...init })
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
): Promise<{ expiresAt: string; user: { id: string; email: string; emailVerified: boolean } }> {
  return request('/api/auth/register', authSessionSchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
}

export async function login(
  email: string,
  password: string,
): Promise<{ expiresAt: string; user: { id: string; email: string; emailVerified: boolean } }> {
  return request('/api/auth/login', authSessionSchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
}

export async function logout(): Promise<void> {
  await request('/api/auth/logout', z.object({ status: z.string() }), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  })
}

export async function getMe(): Promise<{ id: string; email: string; emailVerified: boolean }> {
  return request('/api/auth/me', userProfileSchema)
}

export async function forgotPassword(email: string): Promise<{ message: string }> {
  return request('/api/auth/forgot-password', messageSchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  })
}

export async function resetPassword(token: string, password: string): Promise<{ message: string }> {
  return request('/api/auth/reset-password', messageSchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, password }),
  })
}

export async function verifyEmail(token: string): Promise<{ message: string }> {
  return request('/api/auth/verify-email', messageSchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  })
}

export async function listMyOrders(): Promise<Order[]> {
  const data = await request('/api/orders/mine', z.object({ orders: z.array(orderSchema) }))
  return data.orders
}

export async function getOrderBySession(sessionId: string, accessToken?: string | null): Promise<Order> {
  const headers: Record<string, string> = {}
  if (accessToken) {
    headers['X-Order-Token'] = accessToken
  }
  return request(`/api/orders/by-session/${sessionId}`, orderSchema, { headers })
}
