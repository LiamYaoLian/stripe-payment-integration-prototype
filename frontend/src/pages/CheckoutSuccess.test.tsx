import { act, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { paidOrder } from '../test/fixtures/orders'
import { renderWithRouter } from '../test/renderWithRouter'
import { CheckoutSuccess } from './CheckoutSuccess'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, getOrderBySession: vi.fn() }
})

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

    renderWithRouter(<CheckoutSuccess />, {
      path: '/checkout/success',
      initialEntry: '/checkout/success?session_id=cs_test_abc',
    })
    expect(screen.getByText(/Confirming payment/)).toBeInTheDocument()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })

    expect(screen.getByText('paid')).toBeInTheDocument()
    expect(client.getOrderBySession).toHaveBeenCalledTimes(2)
  })

  it('shows error after max retries', async () => {
    vi.mocked(client.getOrderBySession).mockRejectedValue(new Error('network'))

    renderWithRouter(<CheckoutSuccess />, {
      path: '/checkout/success',
      initialEntry: '/checkout/success?session_id=cs_test_abc',
    })

    await act(async () => {
      for (let i = 0; i < 30; i++) {
        await vi.advanceTimersByTimeAsync(1000)
      }
    })

    expect(screen.getByText('network')).toBeInTheDocument()
  })
})
