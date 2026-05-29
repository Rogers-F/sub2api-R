/**
 * Chat conversation API endpoints (non-admin).
 * Uses the shared JWT axios client. The persisted conversation/message store is
 * separate from the streaming model relay (which keeps using the raw sk- key path).
 * Endpoints are registered under the same authenticated user group as /user/profile.
 */

import { apiClient } from './client'
import type {
  ChatMessage,
  Conversation,
  CreateConversationRequest,
  ListConversationsResponse,
  ListMessagesResponse,
  PersistMessageInput,
  PersistMessagesResponse,
  UpdateConversationRequest
} from '@/types/chat'

/**
 * List conversations for the current user (cursor-paginated, newest first).
 */
export async function listConversations(
  cursor?: string,
  limit = 30
): Promise<ListConversationsResponse> {
  const { data } = await apiClient.get<ListConversationsResponse>('/conversations', {
    params: { cursor, limit }
  })
  return data
}

/**
 * Create a conversation. Idempotent on client_conversation_id, so retries and the
 * lazy "create on first send" flow are safe.
 */
export async function createConversation(
  payload: CreateConversationRequest
): Promise<Conversation> {
  const { data } = await apiClient.post<Conversation>('/conversations', payload)
  return data
}

/**
 * Fetch conversation metadata by id.
 */
export async function getConversation(id: number): Promise<Conversation> {
  const { data } = await apiClient.get<Conversation>(`/conversations/${id}`)
  return data
}

/**
 * Rename a conversation.
 */
export async function updateConversation(
  id: number,
  payload: UpdateConversationRequest
): Promise<Conversation> {
  const { data } = await apiClient.patch<Conversation>(`/conversations/${id}`, payload)
  return data
}

/**
 * Delete a conversation and its messages.
 */
export async function deleteConversation(id: number): Promise<void> {
  await apiClient.delete(`/conversations/${id}`)
}

/**
 * List persisted messages for a conversation (cursor-paginated, oldest first).
 */
export async function listMessages(
  conversationId: number,
  cursor?: string,
  limit = 100
): Promise<ListMessagesResponse> {
  const { data } = await apiClient.get<ListMessagesResponse>(
    `/conversations/${conversationId}/messages`,
    { params: { cursor, limit } }
  )
  return data
}

/**
 * Persist one or more messages (user message before streaming, assistant message
 * once the stream reaches a terminal state). Each carries a client_message_id for
 * idempotency.
 */
export async function persistMessages(
  conversationId: number,
  messages: PersistMessageInput[]
): Promise<ChatMessage[]> {
  const { data } = await apiClient.post<PersistMessagesResponse>(
    `/conversations/${conversationId}/messages`,
    { messages }
  )
  return data.items
}

export const chatAPI = {
  listConversations,
  createConversation,
  getConversation,
  updateConversation,
  deleteConversation,
  listMessages,
  persistMessages
}

export default chatAPI
