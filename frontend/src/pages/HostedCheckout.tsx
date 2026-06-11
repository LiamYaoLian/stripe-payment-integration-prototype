import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { createCheckoutSession } from '../api/client'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { clearIdempotencyKey, getIdempotencyKey } from '../lib/idempotency'

export function HostedCheckout() {
  const [params] = useSearchParams()
  const productId = params.get('productId')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!productId) {
      setError('Missing productId')
      return
    }

    let cancelled = false
    const idempotencyKey = getIdempotencyKey(productId)

    createCheckoutSession(
      { uiMode: 'hosted', items: [{ productId, quantity: 1 }] },
      idempotencyKey,
    )
      .then((result) => {
        if (cancelled) {
          return
        }
        if (!result.url) {
          setError('No checkout URL returned')
          return
        }
        clearIdempotencyKey(productId)
        window.location.href = result.url
      })
      .catch((checkoutError) => {
        if (!cancelled) {
          setError(checkoutError instanceof Error ? checkoutError.message : 'Checkout failed')
        }
      })

    return () => {
      cancelled = true
    }
  }, [productId])

  if (error) {
    return <ErrorMessage message={error} />
  }
  return <LoadingMessage message="Redirecting to Stripe Checkout..." />
}
