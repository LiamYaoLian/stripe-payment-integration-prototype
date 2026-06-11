import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import type { Order } from '../types/api'
import { MyOrders } from './MyOrders'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return {
    ...actual,
    createGuestSession: vi.fn(),
    listMyOrders: vi.fn(),
  }
})

const orders: Order[] = [
  {
    id: 'ord_1',
    orderNumber: 'ORD-001',
    status: 'paid',
    totalAmountCents: 4900,
    currency: 'usd',
    items: [{ productName: 'Pro Plan', quantity: 1, lineTotalCents: 4900 }],
  },
]

describe('MyOrders', () => {
  beforeEach(() => {
    sessionStorage.clear()
    localStorage.clear()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('shows sign-in form and lists orders after guest session', async () => {
    vi.mocked(client.createGuestSession).mockResolvedValue({
      token: 'guest-jwt',
      expiresAt: '2099-01-01T00:00:00Z',
      role: 'guest',
    })
    vi.mocked(client.listMyOrders).mockResolvedValue(orders)

    render(
      <MemoryRouter>
        <MyOrders />
      </MemoryRouter>,
    )

    expect(screen.getByLabelText(/Email/i)).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText(/Email/i), { target: { value: 'buyer@example.com' } })
    fireEvent.change(screen.getByLabelText(/Order ID/i), { target: { value: 'ord_1' } })
    fireEvent.change(screen.getByLabelText(/Access token/i), { target: { value: 'tok_abc' } })
    fireEvent.click(screen.getByRole('button', { name: 'View orders' }))

    await waitFor(() => {
      expect(screen.getByText('ORD-001')).toBeInTheDocument()
    })
    expect(screen.getByText('$49.00')).toBeInTheDocument()
  })
})
