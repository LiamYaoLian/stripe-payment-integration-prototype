import { Route, Routes } from 'react-router-dom'
import Catalog from './pages/Catalog'
import HostedCheckout from './pages/HostedCheckout'
import EmbeddedCheckout from './pages/EmbeddedCheckout'
import CheckoutSuccess from './pages/CheckoutSuccess'
import CheckoutCancel from './pages/CheckoutCancel'
import CheckoutComplete from './pages/CheckoutComplete'

export default function App() {
  return (
    <main>
      <Routes>
        <Route path="/" element={<Catalog />} />
        <Route path="/checkout/hosted" element={<HostedCheckout />} />
        <Route path="/checkout/embedded" element={<EmbeddedCheckout />} />
        <Route path="/checkout/success" element={<CheckoutSuccess />} />
        <Route path="/checkout/cancel" element={<CheckoutCancel />} />
        <Route path="/checkout/complete" element={<CheckoutComplete />} />
      </Routes>
    </main>
  )
}
