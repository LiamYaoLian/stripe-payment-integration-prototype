import { act, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import CheckoutSuccess from './CheckoutSuccess'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, getOrderBySession: vi.fn() }
})

const paidOrder: client.Order = {
  id: 'ord1',
  orderNumber: 'ORD-1',
  status: 'paid',
  totalAmountCents: 4900,
  currency: 'usd',
  items: [{ productName: 'Pro', quantity: 1, lineTotalCents: 4900 }],
}

function renderPage(sessionId = 'cs_test_abc') {
  return render(
    <MemoryRouter initialEntries={[`/checkout/success?session_id=${sessionId}`]}>
      <Routes>
        <Route path="/checkout/success" element={<CheckoutSuccess />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('CheckoutSuccess', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('retries polling after transient errors', async () => {
    vi.mocked(client.getOrderBySession)
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce(paidOrder)

    renderPage()
    expect(screen.getByText(/Confirming payment/)).toBeInTheDocument()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })

    expect(screen.getByText('paid')).toBeInTheDocument()
    expect(client.getOrderBySession).toHaveBeenCalledTimes(2)
  })

  it('shows error after max retries', async () => {
    vi.mocked(client.getOrderBySession).mockRejectedValue(new Error('network'))

    renderPage()

    await act(async () => {
      for (let i = 0; i < 30; i++) {
        await vi.advanceTimersByTimeAsync(1000)
      }
    })

    expect(screen.getByText('network')).toBeInTheDocument()
  })
})
