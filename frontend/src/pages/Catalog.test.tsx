import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import Catalog from './Catalog'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return { ...actual, listProducts: vi.fn() }
})

const products: client.Product[] = [
  { id: 'p1', name: 'Pro Plan', description: 'Best value', unitAmountCents: 4900, currency: 'usd' },
]

describe('Catalog', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders products with checkout links', async () => {
    vi.mocked(client.listProducts).mockResolvedValue(products)

    render(
      <MemoryRouter>
        <Catalog />
      </MemoryRouter>,
    )

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

    render(
      <MemoryRouter>
        <Catalog />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('API down')).toBeInTheDocument()
    })
  })
})
