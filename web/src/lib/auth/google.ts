import { googleClientId } from '../config'

type CredentialCallback = (idToken: string) => void

let callback: CredentialCallback | null = null
let loading: Promise<void> | null = null

function loadScript(): Promise<void> {
  if (loading) return loading
  loading = new Promise((resolve, reject) => {
    if (window.google?.accounts) {
      resolve()
      return
    }
    const script = document.createElement('script')
    script.src = 'https://accounts.google.com/gsi/client'
    script.async = true
    script.defer = true
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('failed to load the Google sign-in script'))
    document.head.appendChild(script)
  })
  return loading
}

export function init(onCredential: CredentialCallback): void {
  if (!googleClientId) {
    throw new Error('Google sign-in is not configured (set VITE_GOOGLE_CLIENT_ID)')
  }
  callback = onCredential
}

export async function start(): Promise<boolean> {
  await loadScript()
  if (!callback) return false
  if (!window.google?.accounts) return false

  window.google.accounts.id.initialize({
    client_id: googleClientId,
    callback: (response) => {
      callback?.(response.credential)
    },
  })

  return true
}

export function renderButton(parent: HTMLElement): void {
  if (!window.google?.accounts) return
  window.google.accounts.id.renderButton(parent, {
    theme: 'outline',
    size: 'large',
    type: 'standard',
    text: 'continue_with',
  })
}