import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { AuthProvider } from '../context/AuthContext'
import type { Product } from '../types/api'
import { Catalog } from './Catalog'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, listProducts: vi.fn(), getMe: vi.fn() }
})

function renderCatalog() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <Catalog />
      </AuthProvider>
    </MemoryRouter>,
  )
}

const products: Product[] = [
  { id: 'p1', name: 'Pro Plan', description: 'Best value', unitAmountCents: 4900, currency: 'usd' },
]

describe('Catalog', () => {
  beforeEach(() => {
    vi.mocked(client.getMe).mockRejectedValue(new client.ApiError(401, 'UNAUTHORIZED', 'not signed in'))
  })

  afterEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('renders products with checkout links', async () => {
    vi.mocked(client.listProducts).mockResolvedValue(products)

    renderCatalog()

    expect(screen.getByText(/Loading products/)).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText('Pro Plan')).toBeInTheDocument()
    })

    expect(screen.getByText('Best value')).toBeInTheDocument()
    expect(screen.getByText('$49.00')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Pay (Hosted)' })).toHaveAttribute(
      'href',
      '/checkout/hosted?productId=p1',
    )
    expect(screen.getByRole('link', { name: 'Pay (Embedded)' })).toHaveAttribute(
      'href',
      '/checkout/embedded?productId=p1',
    )
  })

  it('shows error when product load fails', async () => {
    vi.mocked(client.listProducts).mockRejectedValue(new Error('API down'))

    renderCatalog()

    await waitFor(() => {
      expect(screen.getByText('API down')).toBeInTheDocument()
    })
  })
})
