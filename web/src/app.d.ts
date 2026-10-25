// Minimal types for the Google Identity Services library loaded at runtime
// from https://accounts.google.com/gsi/client.
interface GsiCredentialResponse {
  credential: string
}

interface GsiIdConfiguration {
  client_id: string
  callback: (response: GsiCredentialResponse) => void
}

interface GoogleAccountsId {
  initialize: (config: GsiIdConfiguration) => void
  renderButton: (parent: HTMLElement, options: Record<string, unknown>) => void
}

interface Window {
  google?: {
    accounts: {
      id: GoogleAccountsId
    }
  }
}
