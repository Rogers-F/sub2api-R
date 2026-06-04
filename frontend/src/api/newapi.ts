// new-api BFF path: same-origin endpoints authenticated by the app JWT (NOT an sk- key).
// Uses raw fetch (not axios) so we can stream the SSE body; the JWT is read from
// localStorage the same way the axios client injects it (see api/client.ts).
import type { ChatCompletionPayload, ModelOption } from '@/types/playground'
import {
  consumeSseStream,
  extractError,
  type ChatCompletionError,
  type StreamHandlers
} from '@/api/playground'

const BFF_BASE = '/api/v1/newapi'

// Read the JWT exactly like the axios request interceptor does (api/client.ts:58):
// the token lives in localStorage under 'auth_token' and is kept fresh there by
// the refresh flow, so this stays valid across token rotation.
function getJwt(): string {
  return localStorage.getItem('auth_token') ?? ''
}

export async function listModelsBFF(signal?: AbortSignal): Promise<ModelOption[]> {
  const res = await fetch(`${BFF_BASE}/models`, {
    method: 'GET',
    headers: { Authorization: `Bearer ${getJwt()}` },
    signal
  })
  if (!res.ok) {
    throw await extractError(res)
  }
  const body = await res.json()
  const list: Array<{ id: string; display_name?: string }> = body?.data ?? []
  return list.map((m) => ({
    value: m.id,
    label: m.display_name || m.id
  }))
}

export async function chatCompletionStreamBFF(
  payload: ChatCompletionPayload,
  handlers: StreamHandlers,
  signal?: AbortSignal
): Promise<void> {
  let res: Response
  try {
    res = await fetch(`${BFF_BASE}/chat/completions`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${getJwt()}`,
        'Content-Type': 'application/json',
        Accept: 'text/event-stream'
      },
      body: JSON.stringify({ ...payload, stream: true }),
      signal
    })
  } catch (e: any) {
    handlers.onError({ status: 0, message: e?.message ?? 'network error' })
    return
  }

  await consumeSseStream(res, handlers)
}

export type { ChatCompletionError }
