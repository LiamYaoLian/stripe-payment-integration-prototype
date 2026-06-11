import { BackToCatalogLink } from '../components/BackToCatalogLink'

export function CheckoutCancel() {
  return (
    <div>
      <h1>Checkout Canceled</h1>
      <p>Your payment was not completed.</p>
      <BackToCatalogLink />
    </div>
  )
}
