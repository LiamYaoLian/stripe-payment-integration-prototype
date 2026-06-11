import { useEffect, useState } from 'react'
import { getOrderBySession } from '../api/client'
import { getOrderAccessToken } from '../lib/orderToken'
import {
  POLL_INTERVAL_MS,
  POLL_MAX_ATTEMPTS,
  POLL_TIMEOUT_MESSAGE,
  TERMINAL_ORDER_STATUSES,
} from '../constants/checkout'
import type { Order } from '../types/api'

type OrderPollingState = {
  order: Order | null
  error: string | null
  isLoading: boolean
  timedOut: boolean
}

export function useOrderPolling(sessionId: string | null): OrderPollingState {
  const [order, setOrder] = useState<Order | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(Boolean(sessionId))
  const [timedOut, setTimedOut] = useState(false)

  useEffect(() => {
    if (!sessionId) {
      setError('Missing session_id')
      setIsLoading(false)
      return
    }

    let attempts = 0
    let cancelled = false
    let timeoutId: ReturnType<typeof setTimeout> | undefined

    const schedulePoll = () => {
      timeoutId = setTimeout(poll, POLL_INTERVAL_MS)
    }

    const stopPolling = () => {
      setIsLoading(false)
    }

    const poll = async () => {
      if (cancelled) {
        return
      }
      try {
        const nextOrder = await getOrderBySession(sessionId, getOrderAccessToken(sessionId))
        if (cancelled) {
          return
        }
        setOrder(nextOrder)
        setError(null)
        setTimedOut(false)
        stopPolling()
        if (TERMINAL_ORDER_STATUSES.includes(nextOrder.status)) {
          return
        }
        if (attempts++ < POLL_MAX_ATTEMPTS) {
          setIsLoading(true)
          schedulePoll()
          return
        }
        setTimedOut(true)
      } catch (pollError) {
        if (cancelled) {
          return
        }
        if (attempts++ < POLL_MAX_ATTEMPTS) {
          setIsLoading(true)
          schedulePoll()
          return
        }
        setError(pollError instanceof Error ? pollError.message : 'Failed to load order')
        stopPolling()
      }
    }

    setIsLoading(true)
    setTimedOut(false)
    void poll()

    return () => {
      cancelled = true
      if (timeoutId !== undefined) {
        clearTimeout(timeoutId)
      }
    }
  }, [sessionId])

  return { order, error, isLoading, timedOut }
}
