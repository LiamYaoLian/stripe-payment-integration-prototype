import { useEffect, useState } from 'react'
import { createCheckoutSession } from '../api/client'
import { getIdempotencyKey } from '../lib/idempotency'
import { saveCheckoutContext } from '../lib/checkoutContext'
import { saveOrderAccessToken } from '../lib/orderToken'
import type { CheckoutSessionResult } from '../types/api'

type CreateCheckoutSessionState = {
  session: CheckoutSessionResult | null
  error: string | null
  loading: boolean
}

export function useCreateCheckoutSession(
  productId: string | null,
  uiMode: 'hosted' | 'embedded',
  onBeforeCreate?: (productId: string) => void,
): CreateCheckoutSessionState {
  const [session, setSession] = useState<CheckoutSessionResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(Boolean(productId))

  useEffect(() => {
    if (!productId) {
      setError('Missing productId')
      setLoading(false)
      return
    }

    let cancelled = false
    onBeforeCreate?.(productId)
    const idempotencyKey = getIdempotencyKey(productId, uiMode)

    createCheckoutSession({ uiMode, items: [{ productId, quantity: 1 }] }, idempotencyKey)
      .then((result) => {
        if (!cancelled) {
          if (result.accessToken) {
            saveOrderAccessToken(result.sessionId, result.accessToken)
            saveCheckoutContext({
              orderId: result.orderId,
              sessionId: result.sessionId,
              accessToken: result.accessToken,
            })
          }
          setSession(result)
          setError(null)
        }
      })
      .catch((checkoutError) => {
        if (!cancelled) {
          setError(checkoutError instanceof Error ? checkoutError.message : 'Checkout failed')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [productId, uiMode, onBeforeCreate])

  return { session, error, loading }
}
