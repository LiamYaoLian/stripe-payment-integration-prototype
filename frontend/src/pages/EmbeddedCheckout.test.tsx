import { screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { renderWithRouter } from '../test/renderWithRouter'
import { EmbeddedCheckoutPage } from './EmbeddedCheckout'

vi.mock('@stripe/react-stripe-js', () => ({
  EmbeddedCheckoutProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  EmbeddedCheckout: () => <div data-testid="embedded-checkout">Stripe embedded form</div>,
}))

vi.mock('@stripe/stripe-js', () => ({
  loadStripe: vi.fn(() => Promise.resolve({})),
}))

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, createCheckoutSession: vi.fn() }
})

describe('EmbeddedCheckoutPage', () => {
  afterEach(() => {
    vi.clearAllMocks()
    vi.stubEnv('VITE_STRIPE_PUBLISHABLE_KEY', 'pk_test')
  })

  it('renders embedded checkout when client secret is returned', async () => {
    vi.mocked(client.createCheckoutSession).mockResolvedValue({
      orderId: 'ord1',
      orderNumber: 'ORD-1',
      sessionId: 'cs_test',
      clientSecret: 'cs_test_secret',
    })

    renderWithRouter(<EmbeddedCheckoutPage />, {
      path: '/checkout/embedded',
      initialEntry: '/checkout/embedded?productId=p1',
    })

    await waitFor(() => {
      expect(screen.getByTestId('embedded-checkout')).toBeInTheDocument()
    })
  })

  it('shows error when productId is missing', async () => {
    renderWithRouter(<EmbeddedCheckoutPage />, {
      path: '/checkout/embedded',
      initialEntry: '/checkout/embedded',
    })

    await waitFor(() => {
      expect(screen.getByText('Missing productId')).toBeInTheDocument()
    })
  })
})
