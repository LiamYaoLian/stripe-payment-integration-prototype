import { useSearchParams } from 'react-router-dom'
import { loadStripe } from '@stripe/stripe-js'
import { EmbeddedCheckoutProvider, EmbeddedCheckout } from '@stripe/react-stripe-js'
import { ErrorMessage } from '../components/ErrorMessage'
import { LoadingMessage } from '../components/LoadingMessage'
import { LAST_EMBEDDED_PRODUCT_ID_KEY } from '../constants/checkout'
import { useCreateCheckoutSession } from '../hooks/useCreateCheckoutSession'

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY)

function rememberEmbeddedProduct(productId: string) {
  sessionStorage.setItem(LAST_EMBEDDED_PRODUCT_ID_KEY, productId)
}

export function EmbeddedCheckoutPage() {
  const [params] = useSearchParams()
  const productId = params.get('productId')
  const { session, error, loading } = useCreateCheckoutSession(
    productId,
    'embedded',
    rememberEmbeddedProduct,
  )

  if (!import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY && !error) {
    return <ErrorMessage message="VITE_STRIPE_PUBLISHABLE_KEY is not set" />
  }
  if (error) {
    return <ErrorMessage message={error} />
  }
  if (loading || !session?.clientSecret) {
    return <LoadingMessage message="Loading checkout..." />
  }

  return (
    <div>
      <h1>Embedded Checkout</h1>
      <EmbeddedCheckoutProvider stripe={stripePromise} options={{ clientSecret: session.clientSecret }}>
        <EmbeddedCheckout />
      </EmbeddedCheckoutProvider>
    </div>
  )
}
