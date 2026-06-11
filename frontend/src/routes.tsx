import { Route } from 'react-router-dom'
import { Catalog } from './pages/Catalog'
import { CheckoutCancel } from './pages/CheckoutCancel'
import { CheckoutComplete } from './pages/CheckoutComplete'
import { CheckoutSuccess } from './pages/CheckoutSuccess'
import { EmbeddedCheckoutPage } from './pages/EmbeddedCheckout'
import { HostedCheckout } from './pages/HostedCheckout'
import { MyOrders } from './pages/MyOrders'

export function AppRoutes() {
  return (
    <>
      <Route path="/" element={<Catalog />} />
      <Route path="/orders" element={<MyOrders />} />
      <Route path="/checkout/hosted" element={<HostedCheckout />} />
      <Route path="/checkout/embedded" element={<EmbeddedCheckoutPage />} />
      <Route path="/checkout/success" element={<CheckoutSuccess />} />
      <Route path="/checkout/cancel" element={<CheckoutCancel />} />
      <Route path="/checkout/complete" element={<CheckoutComplete />} />
    </>
  )
}
