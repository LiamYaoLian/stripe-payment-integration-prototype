import { act, cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import * as idempotency from '../lib/idempotency'
import type { Order } from '../types/api'
import { CheckoutComplete } from './CheckoutComplete'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, getOrderBySession: vi.fn() }
})

vi.mock('../lib/idempotency', async () => {
  const actual = await vi.importActual<typeof import('../lib/idempotency')>('../lib/idempotency')
  return { ...actual, clearIdempotencyKey: vi.fn() }
})

const paidOrder: Order = {
  id: 'ord1',
  orderNumber: 'ORD-1',
  status: 'paid',
  totalAmountCents: 4900,
  currency: 'usd',
  items: [{ productName: 'Pro', quantity: 1, lineTotalCents: 4900 }],
}

function renderPage(sessionId?: string) {
  const path = sessionId ? `/checkout/complete?session_id=${sessionId}` : '/checkout/complete'
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/checkout/complete" element={<CheckoutComplete />} />
      </Routes>
    </MemoryRouter>,
  )
}

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

    renderPage('cs_test_embed')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(idempotency.clearIdempotencyKey).toHaveBeenCalledWith('prod-embed')
    expect(screen.getByText('paid')).toBeInTheDocument()
  })

  it('shows error when session_id is missing', async () => {
    renderPage()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(screen.getByText('Missing session_id')).toBeInTheDocument()
  })

  it('retries polling after transient errors', async () => {
    vi.mocked(client.getOrderBySession)
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce(paidOrder)

    renderPage('cs_test_retry')

    expect(screen.getByText(/Confirming payment/)).toBeInTheDocument()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })

    expect(screen.getByText('ORD-1')).toBeInTheDocument()
    expect(screen.getByText('paid')).toBeInTheDocument()
    expect(client.getOrderBySession).toHaveBeenCalledTimes(2)
  })
})
