const ORDER_TOKEN_PREFIX = 'order-access-token:'

export function saveOrderAccessToken(sessionId: string, token: string) {
  sessionStorage.setItem(`${ORDER_TOKEN_PREFIX}${sessionId}`, token)
}

export function getOrderAccessToken(sessionId: string): string | null {
  return sessionStorage.getItem(`${ORDER_TOKEN_PREFIX}${sessionId}`)
}

export function clearOrderAccessToken(sessionId: string) {
  sessionStorage.removeItem(`${ORDER_TOKEN_PREFIX}${sessionId}`)
}
