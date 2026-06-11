import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import type { Order } from '../types/api'
import { AuthProvider } from '../context/AuthContext'
import { MyOrders } from './MyOrders'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return {
    ...actual,
    getMe: vi.fn(),
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

function renderMyOrders() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <MyOrders />
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('MyOrders', () => {
  beforeEach(() => {
    localStorage.setItem('user-session-token', 'user-jwt')
    vi.mocked(client.getMe).mockResolvedValue({
      id: 'cust_1',
      email: 'buyer@example.com',
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('lists orders for a signed-in user', async () => {
    vi.mocked(client.listMyOrders).mockResolvedValue(orders)

    renderMyOrders()

    await waitFor(() => {
      expect(screen.getByText('ORD-001')).toBeInTheDocument()
    })
    expect(screen.getByText('$49.00')).toBeInTheDocument()
    expect(screen.getByText(/buyer@example.com/)).toBeInTheDocument()
  })
})
