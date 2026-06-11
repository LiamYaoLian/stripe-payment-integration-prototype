import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { AuthProvider } from '../context/AuthContext'
import { SignUp } from './SignUp'

const navigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useNavigate: () => navigate,
  }
})

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return {
    ...actual,
    register: vi.fn(),
    getMe: vi.fn(),
  }
})

describe('SignUp', () => {
  beforeEach(() => {
    vi.mocked(client.getMe).mockRejectedValue(new client.ApiError(401, 'UNAUTHORIZED', 'not signed in'))
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('shows error when passwords do not match', async () => {
    render(
      <MemoryRouter>
        <AuthProvider>
          <SignUp />
        </AuthProvider>
      </MemoryRouter>,
    )

    fireEvent.change(screen.getByLabelText(/Email/i), { target: { value: 'buyer@example.com' } })
    fireEvent.change(screen.getByLabelText(/^Password$/i), { target: { value: 'password123' } })
    fireEvent.change(screen.getByLabelText(/Confirm password/i), { target: { value: 'different1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign up' }))

    expect(await screen.findByText('Passwords do not match')).toBeInTheDocument()
    expect(client.register).not.toHaveBeenCalled()
  })

  it('submits registration and navigates on success', async () => {
    vi.mocked(client.register).mockResolvedValue({
      expiresAt: '2099-01-01T00:00:00Z',
      user: { id: 'cust_1', email: 'buyer@example.com', emailVerified: false },
    })

    render(
      <MemoryRouter>
        <AuthProvider>
          <SignUp />
        </AuthProvider>
      </MemoryRouter>,
    )

    fireEvent.change(screen.getByLabelText(/Email/i), { target: { value: 'buyer@example.com' } })
    fireEvent.change(screen.getByLabelText(/^Password$/i), { target: { value: 'password123' } })
    fireEvent.change(screen.getByLabelText(/Confirm password/i), { target: { value: 'password123' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign up' }))

    await waitFor(() => {
      expect(client.register).toHaveBeenCalledWith('buyer@example.com', 'password123')
    })
    expect(navigate).toHaveBeenCalledWith('/orders', { replace: true })
  })
})
