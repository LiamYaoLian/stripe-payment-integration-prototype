const CHECKOUT_CONTEXT_KEY = 'last-checkout-context'

export type CheckoutContext = {
  orderId: string
  sessionId: string
  accessToken: string
}

export function saveCheckoutContext(context: CheckoutContext) {
  sessionStorage.setItem(CHECKOUT_CONTEXT_KEY, JSON.stringify(context))
}

export function getCheckoutContext(): CheckoutContext | null {
  const raw = sessionStorage.getItem(CHECKOUT_CONTEXT_KEY)
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw) as CheckoutContext
    if (parsed.orderId && parsed.sessionId && parsed.accessToken) {
      return parsed
    }
  } catch {
    return null
  }
  return null
}
