import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { loadStripe } from '@stripe/stripe-js'
import { EmbeddedCheckoutProvider, EmbeddedCheckout } from '@stripe/react-stripe-js'
import { createCheckoutSession, getIdempotencyKey } from '../api/client'

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY)

export default function EmbeddedCheckoutPage() {
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

    sessionStorage.setItem('last-embedded-product-id', productId)
    const idempotencyKey = getIdempotencyKey(productId)
    createCheckoutSession(
      { uiMode: 'embedded', items: [{ productId, quantity: 1 }] },
      idempotencyKey,
    )
      .then((result) => {
        if (!result.clientSecret) {
          setError('No client secret returned')
          return
        }
        setClientSecret(result.clientSecret)
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'Checkout failed'))
  }, [productId])

  if (error) return <p className="error">{error}</p>
  if (!clientSecret) return <p>Loading checkout...</p>

  return (
    <div>
      <h1>Embedded Checkout</h1>
      <EmbeddedCheckoutProvider stripe={stripePromise} options={{ clientSecret }}>
        <EmbeddedCheckout />
      </EmbeddedCheckoutProvider>
    </div>
  )
}
