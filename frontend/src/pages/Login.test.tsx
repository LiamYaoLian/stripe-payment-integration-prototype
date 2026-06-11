import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { AuthProvider } from '../context/AuthContext'
import { Login } from './Login'

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
    login: vi.fn(),
    getMe: vi.fn(),
  }
})

describe('Login', () => {
  beforeEach(() => {
    vi.mocked(client.getMe).mockRejectedValue(new client.ApiError(401, 'UNAUTHORIZED', 'not signed in'))
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('submits credentials and navigates on success', async () => {
    vi.mocked(client.login).mockResolvedValue({
      expiresAt: '2099-01-01T00:00:00Z',
      user: { id: 'cust_1', email: 'buyer@example.com', emailVerified: false },
    })

    render(
      <MemoryRouter>
        <AuthProvider>
          <Login />
        </AuthProvider>
      </MemoryRouter>,
    )

    fireEvent.change(screen.getByLabelText(/Email/i), { target: { value: 'buyer@example.com' } })
    fireEvent.change(screen.getByLabelText(/Password/i), { target: { value: 'password123' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    await waitFor(() => {
      expect(client.login).toHaveBeenCalledWith('buyer@example.com', 'password123')
    })
    expect(navigate).toHaveBeenCalledWith('/orders', { replace: true })
  })
})
