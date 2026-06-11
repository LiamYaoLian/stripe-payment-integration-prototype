import { useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { useCreateCheckoutSession } from '../hooks/useCreateCheckoutSession'
import { clearIdempotencyKey } from '../lib/idempotency'

export function HostedCheckout() {
  const [params] = useSearchParams()
  const productId = params.get('productId')
  const { session, error, loading } = useCreateCheckoutSession(productId, 'hosted')

  useEffect(() => {
    if (!session?.url || !productId) {
      return
    }
    clearIdempotencyKey(productId)
    window.location.href = session.url
  }, [session, productId])

  if (error) {
    return <ErrorMessage message={error} />
  }
  if (loading || !session?.url) {
    return <LoadingMessage message="Redirecting to Stripe Checkout..." />
  }
  return <LoadingMessage message="Redirecting to Stripe Checkout..." />
}
