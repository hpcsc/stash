export interface SessionState {
  signedIn: boolean
  googleClientId: string
  canSignOut: boolean
}

interface Envelope {
  successful: boolean
  messages: string[]
  data?: unknown
}

async function request(method: string, body?: object): Promise<Envelope> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  let res: Response
  try {
    res = await fetch('/session', {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new Error('The event browser is not running.')
  }

  let envelope: Envelope
  try {
    envelope = (await res.json()) as Envelope
  } catch {
    throw new Error('The event browser returned an unexpected response.')
  }

  if (!envelope.successful) {
    throw new Error(envelope.messages.join('\n') || 'The request failed.')
  }
  return envelope
}

export async function readSession(): Promise<SessionState> {
  return (await request('GET')).data as SessionState
}

export async function signIn(idToken: string): Promise<SessionState> {
  return (await request('POST', { idToken })).data as SessionState
}

export async function signOut(): Promise<void> {
  await request('DELETE')
}
