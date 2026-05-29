// Types for the persisted multi-conversation chat feature.
// These mirror the JWT-authenticated backend contract (snake_case payloads).

export type ChatRole = 'user' | 'assistant'
export type ChatMessageStatus = 'complete' | 'error'
export type ConversationStatus = string

export interface Conversation {
  // Backend ids are int64, serialized as JSON numbers.
  id: number
  title: string
  model: string
  status: ConversationStatus
  created_at: string
  updated_at: string
  // Independent ordering column: max(message.created_at) for the conversation.
  // Renames no longer bump this, so the list order stays stable.
  last_message_at: string
}

export interface ChatMessage {
  // Backend ids are int64, serialized as JSON numbers.
  id: number
  conversation_id: number
  role: ChatRole
  content: string
  model: string
  status: ChatMessageStatus
  reported_input_tokens?: number
  reported_output_tokens?: number
  created_at: string
}

// ==================== Request / Response payloads ====================

export interface ListConversationsResponse {
  items: Conversation[]
  next_cursor?: string | null
}

export interface CreateConversationRequest {
  // Client-generated UUID; the create endpoint is idempotent on this value.
  client_conversation_id: string
  title?: string
  model?: string
}

export interface UpdateConversationRequest {
  title: string
}

export interface ListMessagesResponse {
  items: ChatMessage[]
  next_cursor?: string | null
}

// One message to persist. client_message_id is a client-generated UUID for idempotency.
export interface PersistMessageInput {
  client_message_id: string
  role: ChatRole
  content: string
  model?: string
  status: ChatMessageStatus
  reported_input_tokens?: number
  reported_output_tokens?: number
  gateway_request_id?: string
}

export interface PersistMessagesRequest {
  messages: PersistMessageInput[]
}

export interface PersistMessagesResponse {
  items: ChatMessage[]
}

// Atomic replace for "regenerate": truncates from the cutoff message (identified by
// exactly one of from_id / from_client_message_id) and inserts the new assistant
// message in a single server transaction. The response is a single message row.
export interface ReplaceMessagePayload {
  from_id?: number
  from_client_message_id?: string
  message: PersistMessageInput
}
