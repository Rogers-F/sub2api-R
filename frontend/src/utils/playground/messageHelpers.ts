import { MESSAGE_ROLES, MESSAGE_STATUS } from '@/constants/playground'
import type {
  MessageContent,
  MessageRole,
  PlaygroundMessage
} from '@/types/playground'

export function generateMessageId(): string {
  const rand = globalThis.crypto?.randomUUID?.()
  if (rand) return rand
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

export function createMessage(
  role: MessageRole,
  content: MessageContent,
  overrides: Partial<PlaygroundMessage> = {}
): PlaygroundMessage {
  return {
    id: generateMessageId(),
    role,
    content,
    status: MESSAGE_STATUS.COMPLETE,
    createdAt: new Date().toISOString(),
    ...overrides
  }
}

export function createLoadingAssistantMessage(): PlaygroundMessage {
  return createMessage(MESSAGE_ROLES.ASSISTANT, '', {
    status: MESSAGE_STATUS.LOADING,
    reasoningContent: '',
    isReasoningExpanded: true,
    isThinkingComplete: false
  })
}

export function getTextContent(message: PlaygroundMessage): string {
  if (typeof message.content === 'string') {
    return message.content
  }
  if (Array.isArray(message.content)) {
    return message.content
      .filter((part): part is { type: 'text'; text: string } => part.type === 'text')
      .map((part) => part.text)
      .join('\n')
  }
  return ''
}

export function buildMessageContent(
  text: string,
  imageUrls: string[],
  imageEnabled: boolean
): MessageContent {
  const cleanUrls = imageEnabled ? imageUrls.filter((u) => u.trim() !== '') : []
  if (cleanUrls.length === 0) return text
  return [
    { type: 'text', text },
    ...cleanUrls.map((url) => ({
      type: 'image_url' as const,
      image_url: { url }
    }))
  ]
}
