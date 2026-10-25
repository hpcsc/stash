import { googleClientId } from '../config'

type CredentialCallback = (idToken: string) => void

let callback: CredentialCallback | null = null

function loadScript(): Promise<void> {
  return new Promise((resolve, reject) => {
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
}

export async function initGoogleSignIn(onCredential: CredentialCallback): Promise<void> {
  if (!googleClientId) {
    throw new Error('Google sign-in is not configured (set VITE_GOOGLE_CLIENT_ID)')
  }
  callback = onCredential
  await loadScript()
  window.google!.accounts.id.initialize({
    client_id: googleClientId,
    callback: (response) => callback?.(response.credential),
  })
}

export function renderGoogleButton(parent: HTMLElement): void {
  window.google!.accounts.id.renderButton(parent, {
    theme: 'outline',
    size: 'large',
    type: 'standard',
    text: 'continue_with',
  })
}
