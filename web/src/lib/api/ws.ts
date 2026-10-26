import { wsBaseUrl } from '../config'

type MessageHandler = (msg: ServerMessage) => void

interface ServerMessage {
  type: string
  data: unknown
}

let ws: WebSocket | null = null
let backoff = 1000
const maxBackoff = 30000
const handlers = new Set<MessageHandler>()

export function connect(token: string): void {
  if (ws && ws.readyState === WebSocket.OPEN) return

  const url = `${wsBaseUrl}/ws?token=${encodeURIComponent(token)}`
  ws = new WebSocket(url)

  ws.onopen = () => {
    backoff = 1000
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data) as ServerMessage
      for (const handler of handlers) {
        handler(msg)
      }
    } catch {
      // ignore malformed messages
    }
  }

  ws.onclose = () => {
    ws = null
    setTimeout(() => connect(token), backoff)
    backoff = Math.min(backoff * 2, maxBackoff)
  }
}

export function onMessage(handler: MessageHandler): () => void {
  handlers.add(handler)
  return () => handlers.delete(handler)
}

export function disconnect(): void {
  if (ws) {
    ws.onclose = null // prevent reconnect
    ws.close()
    ws = null
  }
}