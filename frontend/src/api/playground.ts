// 不走 axios：relay 端点 /v1/* 要 Bearer sk-...，而 axios 已固定注入 Bearer <jwt>
import type { ChatCompletionPayload, ModelOption } from '@/types/playground'

const RELAY_BASE = '/v1'

export interface ChatCompletionError {
  status: number
  message: string
  type?: string
  raw?: unknown
}

export async function listModels(apiKey: string, signal?: AbortSignal): Promise<ModelOption[]> {
  const res = await fetch(`${RELAY_BASE}/models`, {
    method: 'GET',
    headers: { Authorization: `Bearer ${apiKey}` },
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

export async function chatCompletion(
  apiKey: string,
  payload: ChatCompletionPayload,
  signal?: AbortSignal
): Promise<unknown> {
  const res = await fetch(`${RELAY_BASE}/chat/completions`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ ...payload, stream: false }),
    signal
  })
  if (!res.ok) {
    throw await extractError(res)
  }
  return res.json()
}

export interface StreamHandlers {
  onChunk: (rawJson: string, parsed: unknown) => void
  onDone: () => void
  onError: (err: ChatCompletionError) => void
}

// Drive the OpenAI-style SSE response: parse `data:` frames, dispatch chunks, and
// report the terminal outcome. Shared by both the sub2api gateway path (sk- key)
// and the new-api BFF path (JWT) so the parsing loop stays identical.
export async function consumeSseStream(res: Response, handlers: StreamHandlers): Promise<void> {
  if (!res.ok) {
    handlers.onError(await extractError(res))
    return
  }

  const reader = res.body?.getReader()
  if (!reader) {
    handlers.onError({ status: 0, message: 'No response body' })
    return
  }

  const decoder = new TextDecoder()
  let buffer = ''
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''
      for (const line of lines) {
        const trimmed = line.trim()
        if (!trimmed.startsWith('data:')) continue
        const data = trimmed.slice(5).trim()
        if (!data) continue
        if (data === '[DONE]') {
          handlers.onDone()
          return
        }
        try {
          const parsed = JSON.parse(data)
          handlers.onChunk(data, parsed)
        } catch {
          // 单帧 JSON 异常不应中断整个流
        }
      }
    }
    handlers.onDone()
  } catch (e: any) {
    if (e?.name === 'AbortError') {
      handlers.onDone()
      return
    }
    handlers.onError({ status: 0, message: e?.message ?? 'stream interrupted' })
  }
}

export async function chatCompletionStream(
  apiKey: string,
  payload: ChatCompletionPayload,
  handlers: StreamHandlers,
  signal?: AbortSignal
): Promise<void> {
  let res: Response
  try {
    res = await fetch(`${RELAY_BASE}/chat/completions`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
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

export async function extractError(res: Response): Promise<ChatCompletionError> {
  let body: any = null
  try {
    body = await res.json()
  } catch {
    /* ignore parse */
  }
  const err = body?.error
  return {
    status: res.status,
    message: err?.message || res.statusText || `HTTP ${res.status}`,
    type: err?.type,
    raw: body
  }
}
