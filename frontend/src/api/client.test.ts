import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, clearIdempotencyKey, createCheckoutSession, getIdempotencyKey, listProducts } from './client'

describe('idempotency helpers', () => {
  beforeEach(() => {
    sessionStorage.clear()
    vi.stubGlobal('crypto', { randomUUID: () => 'test-uuid-1' })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('generates and reuses idempotency key per product', () => {
    expect(getIdempotencyKey('prod-a')).toBe('test-uuid-1')
    expect(getIdempotencyKey('prod-a')).toBe('test-uuid-1')
    expect(getIdempotencyKey('prod-b')).toBe('test-uuid-1')
  })

  it('clears idempotency key', () => {
    getIdempotencyKey('prod-a')
    clearIdempotencyKey('prod-a')
    expect(sessionStorage.getItem('checkout-idempotency-prod-a')).toBeNull()
  })
})

describe('listProducts', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('returns products from API envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          data: { products: [{ id: 'p1', name: 'Pro', unitAmountCents: 4900, currency: 'usd' }] },
          error: null,
        }),
      }),
    )

    const products = await listProducts()
    expect(products).toHaveLength(1)
    expect(products[0].name).toBe('Pro')
  })

  it('throws ApiError on API error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({
          data: null,
          error: { code: 'IDEMPOTENCY_CONFLICT', message: 'conflict' },
        }),
      }),
    )

    await expect(
      createCheckoutSession({ uiMode: 'hosted', items: [{ productId: 'p1', quantity: 1 }] }, 'key'),
    ).rejects.toMatchObject({ code: 'IDEMPOTENCY_CONFLICT', status: 409 })
  })

  it('throws on network failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    await expect(listProducts()).rejects.toThrow(/Cannot reach API/)
  })
})

describe('ApiError', () => {
  it('exposes code and status', () => {
    const err = new ApiError(400, 'VALIDATION_ERROR', 'bad input')
    expect(err.message).toBe('bad input')
    expect(err.code).toBe('VALIDATION_ERROR')
    expect(err.status).toBe(400)
  })
})
