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

describe('createCheckoutSession', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('retries on CHECKOUT_IN_PROGRESS then succeeds', async () => {
    const success = {
      orderId: 'ord1',
      orderNumber: 'ORD-1',
      sessionId: 'cs_test',
      url: 'https://checkout.stripe.com/test',
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: false,
        status: 409,
        json: async () => ({
          data: null,
          error: { code: 'CHECKOUT_IN_PROGRESS', message: 'wait' },
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: success, error: null }),
      })
    vi.stubGlobal('fetch', fetchMock)

    const promise = createCheckoutSession(
      { uiMode: 'hosted', items: [{ productId: 'p1', quantity: 1 }] },
      'idem-key',
    )
    await vi.advanceTimersByTimeAsync(500)
    const result = await promise

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(result).toEqual(success)
  })

  it('stops retrying after max attempts on CHECKOUT_IN_PROGRESS', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      json: async () => ({
        data: null,
        error: { code: 'CHECKOUT_IN_PROGRESS', message: 'wait' },
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const promise = createCheckoutSession(
      { uiMode: 'hosted', items: [{ productId: 'p1', quantity: 1 }] },
      'idem-key',
    )
    const assertion = expect(promise).rejects.toMatchObject({ code: 'CHECKOUT_IN_PROGRESS' })
    for (let i = 0; i < 5; i++) {
      await vi.advanceTimersByTimeAsync(500)
    }
    await assertion
    expect(fetchMock).toHaveBeenCalledTimes(5)
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
