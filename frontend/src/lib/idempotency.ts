export type CheckoutUIMode = 'hosted' | 'embedded'

const idempotencyStoragePrefix = 'checkout-idempotency-'

function storageKey(productId: string, uiMode: CheckoutUIMode): string {
  return `${idempotencyStoragePrefix}${uiMode}-${productId}`
}

export function getIdempotencyKey(productId: string, uiMode: CheckoutUIMode): string {
  const key = storageKey(productId, uiMode)
  let value = sessionStorage.getItem(key)
  if (!value) {
    value = crypto.randomUUID()
    sessionStorage.setItem(key, value)
  }
  return value
}

export function clearIdempotencyKey(productId: string, uiMode: CheckoutUIMode): void {
  sessionStorage.removeItem(storageKey(productId, uiMode))
}
