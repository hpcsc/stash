export const apiBaseUrl: string =
  import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:3333'

export const apiPrefix = '/api'

// When the API base is empty the app runs same-origin (nginx serves it and proxies the API),
// so the WebSocket must come from the page origin rather than an http→ws rewrite of the base.
function resolveWsBaseUrl(): string {
  if (apiBaseUrl !== '') return apiBaseUrl.replace(/^http/, 'ws')
  const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${scheme}://${window.location.host}`
}

export const wsBaseUrl: string = resolveWsBaseUrl()

export const googleClientId: string = import.meta.env.VITE_GOOGLE_CLIENT_ID ?? ''
