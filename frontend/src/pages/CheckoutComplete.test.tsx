import { act, cleanup, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import * as idempotency from '../lib/idempotency'
import { paidOrder } from '../test/fixtures/orders'
import { renderWithRouter } from '../test/renderWithRouter'
import { CheckoutComplete } from './CheckoutComplete'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, getOrderBySession: vi.fn() }
})

vi.mock('../lib/idempotency', async () => {
  const actual = await vi.importActual<typeof import('../lib/idempotency')>('../lib/idempotency')
  return { ...actual, clearIdempotencyKey: vi.fn() }
})

describe('CheckoutComplete', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    sessionStorage.clear()
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('clears idempotency key for embedded checkout', async () => {
    sessionStorage.setItem('last-embedded-product-id', 'prod-embed')
    vi.mocked(client.getOrderBySession).mockResolvedValue(paidOrder)

    renderWithRouter(<CheckoutComplete />, {
      path: '/checkout/complete',
      initialEntry: '/checkout/complete?session_id=cs_test_embed',
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(idempotency.clearIdempotencyKey).toHaveBeenCalledWith('prod-embed')
    expect(screen.getByText('paid')).toBeInTheDocument()
  })

  it('shows error when session_id is missing', async () => {
    renderWithRouter(<CheckoutComplete />, {
      path: '/checkout/complete',
      initialEntry: '/checkout/complete',
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(screen.getByText('Missing session_id')).toBeInTheDocument()
  })
})
