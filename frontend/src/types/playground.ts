import type { MESSAGE_ROLES, MESSAGE_STATUS, STREAM_STATUS } from '@/constants/playground'

export type MessageRole = (typeof MESSAGE_ROLES)[keyof typeof MESSAGE_ROLES]
export type MessageStatus = (typeof MESSAGE_STATUS)[keyof typeof MESSAGE_STATUS]
export type StreamStatus = (typeof STREAM_STATUS)[keyof typeof STREAM_STATUS]

interface MessageImageUrl {
  type: 'image_url'
  image_url: { url: string }
}

interface MessageTextPart {
  type: 'text'
  text: string
}

export type MessageContent = string | Array<MessageTextPart | MessageImageUrl>

export interface PlaygroundMessage {
  id: string
  role: MessageRole
  content: MessageContent
  status?: MessageStatus
  reasoningContent?: string
  isReasoningExpanded?: boolean
  isThinkingComplete?: boolean
  createdAt: string
  errorMessage?: string
}

export interface PlaygroundInputs {
  group: string
  apiKeyId: number | null
  model: string
  temperature: number
  top_p: number
  max_tokens: number
  frequency_penalty: number
  presence_penalty: number
  seed: number | null
  stream: boolean
  imageEnabled: boolean
  imageUrls: string[]
}

export interface ParameterEnabled {
  temperature: boolean
  top_p: boolean
  max_tokens: boolean
  frequency_penalty: boolean
  presence_penalty: boolean
  seed: boolean
}

export interface ModelOption {
  value: string
  label: string
}

export interface ChatCompletionPayload {
  model: string
  messages: Array<{ role: MessageRole; content: MessageContent }>
  stream: boolean
  temperature?: number
  top_p?: number
  max_tokens?: number
  frequency_penalty?: number
  presence_penalty?: number
  seed?: number
}
