export const MESSAGE_ROLES = {
  USER: 'user',
  ASSISTANT: 'assistant',
  SYSTEM: 'system'
} as const

export const MESSAGE_STATUS = {
  LOADING: 'loading',
  INCOMPLETE: 'incomplete',
  COMPLETE: 'complete',
  ERROR: 'error'
} as const

export const STREAM_STATUS = {
  IDLE: 'idle',
  STREAMING: 'streaming',
  COMPLETE: 'complete',
  ERROR: 'error'
} as const

export const STORAGE_KEYS = {
  CONFIG: 'playground_config',
  MESSAGES: 'playground_messages'
} as const

export const DEFAULT_INPUTS = {
  group: '',
  apiKeyId: null as number | null,
  model: '',
  temperature: 0.7,
  top_p: 1,
  max_tokens: 4096,
  frequency_penalty: 0,
  presence_penalty: 0,
  seed: null as number | null,
  stream: true,
  imageEnabled: false,
  imageUrls: [''] as string[]
}

export const PLAYGROUND_KEY_EXPIRY_DAYS = 7

export const DEFAULT_PARAMETER_ENABLED = {
  temperature: true,
  top_p: true,
  max_tokens: false,
  frequency_penalty: true,
  presence_penalty: true,
  seed: false
}
