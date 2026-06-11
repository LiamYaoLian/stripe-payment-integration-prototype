import { Route } from 'react-router-dom'
import { ProtectedRoute } from './components/ProtectedRoute'
import { Catalog } from './pages/Catalog'
import { CheckoutCancel } from './pages/CheckoutCancel'
import { CheckoutComplete } from './pages/CheckoutComplete'
import { CheckoutSuccess } from './pages/CheckoutSuccess'
import { EmbeddedCheckoutPage } from './pages/EmbeddedCheckout'
import { HostedCheckout } from './pages/HostedCheckout'
import { Login } from './pages/Login'
import { MyOrders } from './pages/MyOrders'
import { SignUp } from './pages/SignUp'

export function AppRoutes() {
  return (
    <>
      <Route path="/" element={<Catalog />} />
      <Route path="/login" element={<Login />} />
      <Route path="/signup" element={<SignUp />} />
      <Route
        path="/orders"
        element={
          <ProtectedRoute>
            <MyOrders />
          </ProtectedRoute>
        }
      />
      <Route path="/checkout/hosted" element={<HostedCheckout />} />
      <Route path="/checkout/embedded" element={<EmbeddedCheckoutPage />} />
      <Route path="/checkout/success" element={<CheckoutSuccess />} />
      <Route path="/checkout/cancel" element={<CheckoutCancel />} />
      <Route path="/checkout/complete" element={<CheckoutComplete />} />
    </>
  )
}
