import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { clearIdempotencyKey, createCheckoutSession, getIdempotencyKey } from '../api/client'

export default function HostedCheckout() {
  const [params] = useSearchParams()
  const productId = params.get('productId')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!productId) {
      setError('Missing productId')
      return
    }

    const idempotencyKey = getIdempotencyKey(productId)

    createCheckoutSession(
      { uiMode: 'hosted', items: [{ productId, quantity: 1 }] },
      idempotencyKey,
    )
      .then((result) => {
        if (!result.url) {
          setError('No checkout URL returned')
          return
        }
        clearIdempotencyKey(productId)
        window.location.href = result.url
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'Checkout failed'))
  }, [productId])

  if (error) return <p className="error">{error}</p>
  return <p>Redirecting to Stripe Checkout...</p>
}
