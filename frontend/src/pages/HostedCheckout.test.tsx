import { screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import * as idempotency from '../lib/idempotency'
import { renderWithRouter } from '../test/renderWithRouter'
import { HostedCheckout } from './HostedCheckout'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, createCheckoutSession: vi.fn() }
})

vi.mock('../lib/idempotency', async () => {
  const actual = await vi.importActual<typeof import('../lib/idempotency')>('../lib/idempotency')
  return { ...actual, getIdempotencyKey: vi.fn(() => 'idem-1'), clearIdempotencyKey: vi.fn() }
})

describe('HostedCheckout', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('redirects to stripe checkout url', async () => {
    const assign = vi.fn()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { href: 'http://localhost/', assign },
    })

    vi.mocked(client.createCheckoutSession).mockResolvedValue({
      orderId: 'ord1',
      orderNumber: 'ORD-1',
      sessionId: 'cs_test',
      url: 'https://checkout.stripe.com/test',
    })

    renderWithRouter(<HostedCheckout />, {
      path: '/checkout/hosted',
      initialEntry: '/checkout/hosted?productId=p1',
    })

    await waitFor(() => {
      expect(idempotency.clearIdempotencyKey).toHaveBeenCalledWith('p1', 'hosted')
    })
    expect(window.location.href).toBe('https://checkout.stripe.com/test')
  })

  it('shows error when productId is missing', async () => {
    renderWithRouter(<HostedCheckout />, {
      path: '/checkout/hosted',
      initialEntry: '/checkout/hosted',
    })

    await waitFor(() => {
      expect(screen.getByText('Missing productId')).toBeInTheDocument()
    })
  })
})
