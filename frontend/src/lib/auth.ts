const USER_TOKEN_KEY = 'user-session-token'

export function saveUserToken(token: string) {
  localStorage.setItem(USER_TOKEN_KEY, token)
}

export function getUserToken(): string | null {
  return localStorage.getItem(USER_TOKEN_KEY)
}

export function clearUserToken() {
  localStorage.removeItem(USER_TOKEN_KEY)
}
