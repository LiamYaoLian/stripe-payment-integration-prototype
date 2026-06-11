const idempotencyStoragePrefix = 'checkout-idempotency-'

function storageKey(productId: string): string {
  return `${idempotencyStoragePrefix}${productId}`
}

export function getIdempotencyKey(productId: string): string {
  const key = storageKey(productId)
  let value = sessionStorage.getItem(key)
  if (!value) {
    value = crypto.randomUUID()
    sessionStorage.setItem(key, value)
  }
  return value
}

export function clearIdempotencyKey(productId: string): void {
  sessionStorage.removeItem(storageKey(productId))
}
