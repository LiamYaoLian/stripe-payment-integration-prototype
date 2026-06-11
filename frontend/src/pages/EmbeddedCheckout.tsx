import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { loadStripe } from '@stripe/stripe-js'
import { EmbeddedCheckoutProvider, EmbeddedCheckout } from '@stripe/react-stripe-js'
import { createCheckoutSession } from '../api/client'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { LAST_EMBEDDED_PRODUCT_ID_KEY } from '../constants/checkout'
import { getIdempotencyKey } from '../lib/idempotency'

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY)

export function EmbeddedCheckoutPage() {
  const [params] = useSearchParams()
  const productId = params.get('productId')
  const [clientSecret, setClientSecret] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!productId) {
      setError('Missing productId')
      return
    }
    if (!import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY) {
      setError('VITE_STRIPE_PUBLISHABLE_KEY is not set')
      return
    }

    let cancelled = false
    sessionStorage.setItem(LAST_EMBEDDED_PRODUCT_ID_KEY, productId)
    const idempotencyKey = getIdempotencyKey(productId)

    createCheckoutSession(
      { uiMode: 'embedded', items: [{ productId, quantity: 1 }] },
      idempotencyKey,
    )
      .then((result) => {
        if (cancelled) {
          return
        }
        if (!result.clientSecret) {
          setError('No client secret returned')
          return
        }
        setClientSecret(result.clientSecret)
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
  if (!clientSecret) {
    return <LoadingMessage message="Loading checkout..." />
  }

  return (
    <div>
      <h1>Embedded Checkout</h1>
      <EmbeddedCheckoutProvider stripe={stripePromise} options={{ clientSecret }}>
        <EmbeddedCheckout />
      </EmbeddedCheckoutProvider>
    </div>
  )
}
