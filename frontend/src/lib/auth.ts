const GUEST_TOKEN_KEY = 'guest-session-token'

export function saveGuestToken(token: string) {
  sessionStorage.setItem(GUEST_TOKEN_KEY, token)
}

export function getGuestToken(): string | null {
  return sessionStorage.getItem(GUEST_TOKEN_KEY)
}

export function clearGuestToken() {
  sessionStorage.removeItem(GUEST_TOKEN_KEY)
}
