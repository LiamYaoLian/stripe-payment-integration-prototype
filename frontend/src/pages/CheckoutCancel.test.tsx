import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { CheckoutCancel } from './CheckoutCancel'

describe('CheckoutCancel', () => {
  it('shows cancel message and back link', () => {
    render(
      <MemoryRouter>
        <CheckoutCancel />
      </MemoryRouter>,
    )

    expect(screen.getByText('Checkout Canceled')).toBeInTheDocument()
    expect(screen.getByText('Your payment was not completed.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to catalog' })).toHaveAttribute('href', '/')
  })
})
