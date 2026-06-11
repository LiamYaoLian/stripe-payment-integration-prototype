import { Link } from 'react-router-dom'

export default function CheckoutCancel() {
  return (
    <div>
      <h1>Checkout Canceled</h1>
      <p>Your payment was not completed.</p>
      <Link className="btn" to="/">Back to catalog</Link>
    </div>
  )
}
