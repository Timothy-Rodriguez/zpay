// Centralized API client with token-based authentication and refresh interceptor.
//
// Token flow:
//  1. User logs in → backend returns access_token in response body + refresh_token in HttpOnly cookie
//  2. Frontend stores access_token in localStorage via setAccessToken()
//  3. Every protected request automatically includes: Authorization: Bearer <accessToken>
//  4. On 401 response:
//     - Call GET /refresh with credentials: 'include' (browser auto-includes refresh_token cookie)
//     - Backend validates cookie and returns new access_token in response
//     - Frontend updates access_token via setAccessToken() and retries the original request
//  5. refresh_token stays in HttpOnly cookie (browser-managed, never touched by JS)

import { API_BASE_URL } from '../config.js'

const ACCESS_TOKEN_KEY = 'zpay_access_token'
const REFRESH_PATH = '/refresh'

// --- Token storage ---------------------------------------------------------
// Access token is stored in localStorage so it persists across page reloads.
// The refresh_token remains in an HttpOnly cookie (backend-managed, not accessible to JS).
// This variable holds the current access token in memory for fast lookups.

let accessToken = (() => {
  try {
    return localStorage.getItem(ACCESS_TOKEN_KEY) || null
  } catch {
    return null
  }
})()

const tokenListeners = new Set()

export function getAccessToken() {
  return accessToken
}

export function setAccessToken(token) {
  accessToken = token || null
  try {
    if (accessToken) localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
    else localStorage.removeItem(ACCESS_TOKEN_KEY)
  } catch {
    // ignore storage errors (private mode, etc.)
  }
  tokenListeners.forEach((cb) => {
    try {
      cb(accessToken)
    } catch {
      // ignore listener errors
    }
  })
}

export function onAccessTokenChange(cb) {
  tokenListeners.add(cb)
  return () => tokenListeners.delete(cb)
}

// --- Refresh coordination --------------------------------------------------
// When a 401 is received, we need to refresh the access token. The refresh_token
// is automatically included by the browser in the refresh request (via the HttpOnly
// cookie and credentials: 'include'). The backend validates the refresh_token cookie,
// generates a new access_token, and returns it in the response body. We then store
// it and retry the original request with the new token.
// This ensures only one refresh is in flight at a time (deduplication).
let refreshInFlight = null

async function refreshAccessToken() {
  if (refreshInFlight) return refreshInFlight

  refreshInFlight = (async () => {
    const url = `${API_BASE_URL}${REFRESH_PATH}`
    const res = await fetch(url, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })

    if (!res.ok) {
      setAccessToken(null)
      if (res.status === 400 || res.status === 401) {
        window.location.replace('/')
      }
      const err = new Error('Session expired, please log in again')
      err.status = res.status
      throw err
    }

    const data = await res.json().catch(() => ({}))
    const newToken = data.access_token
    if (!newToken) {
      setAccessToken(null)
      throw new Error('Refresh response did not contain access_token')
    }
    setAccessToken(newToken)
    return newToken
  })().finally(() => {
    refreshInFlight = null
  })

  return refreshInFlight
}

// --- Core request with interceptor ----------------------------------------

function buildInit({ method = 'GET', body, headers = {}, ...rest }, token) {
  // Strip internal flags so they don't leak into fetch().
  const cleaned = { ...rest }
  delete cleaned._skipAuth
  delete cleaned._skipRefresh

  const init = {
    method,
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    ...cleaned,
  }
  if (body !== undefined) {
    init.body = typeof body === 'string' ? body : JSON.stringify(body)
  }
  return init
}

async function parseResponse(res) {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return { raw: text }
  }
}

async function request(path, options = {}) {
  const url = `${API_BASE_URL}${path.startsWith('/') ? path : `/${path}`}`
  const skipAuth = options._skipAuth === true
  const skipRefresh = options._skipRefresh === true || path === REFRESH_PATH

  // First attempt with current token.
  let res
  try {
    res = await fetch(url, buildInit(options, skipAuth ? null : accessToken))
  } catch (err) {
    throw new Error(`Network error: ${err.message}`)
  }

  // 401 interceptor: try to refresh once, then replay the original request.
  if (res.status === 401 && !skipAuth && !skipRefresh) {
    try {
      const newToken = await refreshAccessToken()
      res = await fetch(url, buildInit(options, newToken))
    } catch {
      // refresh failed — fall through with the original 401 response.
    }
  }

  const data = await parseResponse(res)

  if (!res.ok) {
    const message =
      (data && (data.error || data.message)) ||
      `Request failed with status ${res.status}`
    const error = new Error(message)
    error.status = res.status
    error.data = data
    throw error
  }
  return data
}

// --- Helpers ---------------------------------------------------------------

export function newIdempotencyKey() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  return `idem-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

// --- Public API ------------------------------------------------------------

export const api = {
  // Auth / users — these endpoints do not need a bearer token.
  signup: (email, password) =>
    request('/signup', {
      method: 'POST',
      body: { email, password },
      _skipAuth: true,
    }),

  login: async (email, password) => {
    const data = await request('/login', {
      method: 'POST',
      body: { email, password },
      _skipAuth: true,
    })
    if (data && data.access_token) {
      setAccessToken(data.access_token)
    }
    return data
  },

  refresh: () =>
    request(REFRESH_PATH, {
      method: 'GET',
      _skipAuth: true,
      _skipRefresh: true,
    }),

  logout: () =>
    request('/logout', { method: 'POST' }).finally(() => {
      setAccessToken(null)
    }),

  // Payments — protected; interceptor will attach the bearer token.
  pay: ({ toEmail, amount, idempotencyKey }) =>
    request('/payment', {
      method: 'POST',
      headers: { 'X-IDEMPOTENCY-KEY': idempotencyKey || newIdempotencyKey() },
      body: { to_email: toEmail, amount: String(amount) },
    }),

  // Balance and transactions — protected.
  getBalance: () => request('/get-balance', { method: 'GET' }),

  getTransactions: () => request('/get-transactions', { method: 'GET' }),

  getDashboard: () => request('/dashboard', { method: 'GET' }),

  checkUserExists: (email) =>
    request(`/user?email=${encodeURIComponent(email)}`, {
      method: 'GET',
      _skipAuth: true,
    }),

  getAccounts: () =>
    request('/get-accounts', {
      method: 'GET',
      _skipAuth: true,
    }),

  // Low-level escape hatch for ad-hoc calls — interceptor still applies.
  request,
}

export default api
