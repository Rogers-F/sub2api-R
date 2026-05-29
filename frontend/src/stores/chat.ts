import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { chatAPI } from '@/api/chat'
import { chatCompletionStream } from '@/api/playground'
import { usePlaygroundStore } from '@/stores/playground'
import { useAppStore } from '@/stores/app'
import { i18n } from '@/i18n'
import { generateMessageId } from '@/utils/playground/messageHelpers'
import type {
  ChatMessage,
  ChatMessageStatus,
  Conversation,
  PersistMessageInput
} from '@/types/chat'

const { t } = i18n.global

// Live (in-memory) representation of a message while streaming. Persisted history is
// the source of truth on reload; we only save once a terminal state is reached.
//
// `id` is always a stable client UUID: it doubles as the v-for key and the
// client_message_id sent to the API, so it must never be replaced with the
// server's numeric id (doing so breaks key stability and string===string lookups).
// The server's numeric id, when needed, is kept separately in `serverId`.
export interface LiveMessage {
  id: string
  serverId?: number
  role: 'user' | 'assistant'
  content: string
  status: ChatMessageStatus | 'streaming'
  errorMessage?: string
  createdAt: string
}

export type ChatStreamStatus = 'idle' | 'streaming' | 'error'

const TITLE_MAX_LENGTH = 40

function truncateTitle(text: string): string {
  const trimmed = text.trim().replace(/\s+/g, ' ')
  if (trimmed.length <= TITLE_MAX_LENGTH) return trimmed
  return `${trimmed.slice(0, TITLE_MAX_LENGTH)}…`
}

function toLiveMessage(m: ChatMessage): LiveMessage {
  // Persisted rows have no client UUID, so derive a stable string key from the
  // numeric server id and retain the numeric value in serverId.
  return {
    id: `srv-${m.id}`,
    serverId: m.id,
    role: m.role,
    content: m.content,
    status: m.status,
    createdAt: m.created_at
  }
}

export const useChatStore = defineStore('chat', () => {
  const playground = usePlaygroundStore()
  const appStore = useAppStore()

  const conversations = ref<Conversation[]>([])
  const conversationsLoading = ref(false)
  const conversationsLoaded = ref(false)
  // Cursor for the next page of conversations (null once the list is exhausted).
  const nextConversationsCursor = ref<string | null>(null)
  const conversationsLoadingMore = ref(false)

  const currentConversationId = ref<number | null>(null)
  const messages = ref<LiveMessage[]>([])
  const messagesLoading = ref(false)
  // Cursor (smallest loaded message id) for fetching an older page; null when the
  // start of history has been reached.
  const messagesPrevCursor = ref<string | null>(null)
  const messagesLoadingMore = ref(false)

  const status = ref<ChatStreamStatus>('idle')
  const abortController = ref<AbortController | null>(null)

  const isStreaming = computed(() => status.value === 'streaming')

  // ==================== Conversation list ====================

  async function loadConversations(): Promise<void> {
    conversationsLoading.value = true
    try {
      const res = await chatAPI.listConversations()
      conversations.value = res.items ?? []
      nextConversationsCursor.value = res.next_cursor ?? null
      conversationsLoaded.value = true
    } catch {
      appStore.showError(t('chat.errors.loadConversationsFailed'))
    } finally {
      conversationsLoading.value = false
    }
  }

  // Fetch the next page of conversations and append it (deduplicated by id).
  async function loadMoreConversations(): Promise<void> {
    const cursor = nextConversationsCursor.value
    if (!cursor || conversationsLoadingMore.value) return
    conversationsLoadingMore.value = true
    try {
      const res = await chatAPI.listConversations(cursor)
      const existing = new Set(conversations.value.map((c) => c.id))
      const fresh = (res.items ?? []).filter((c) => !existing.has(c.id))
      conversations.value = [...conversations.value, ...fresh]
      nextConversationsCursor.value = res.next_cursor ?? null
    } catch {
      appStore.showError(t('chat.errors.loadConversationsFailed'))
    } finally {
      conversationsLoadingMore.value = false
    }
  }

  // Lazily create a conversation: a row is only persisted on the first send.
  async function createConversation(firstUserText: string): Promise<Conversation | null> {
    const clientId = generateMessageId()
    try {
      const conv = await chatAPI.createConversation({
        client_conversation_id: clientId,
        title: truncateTitle(firstUserText),
        model: playground.inputs.model || undefined
      })
      // Idempotent endpoint may return an existing row; de-duplicate by id.
      if (!conversations.value.some((c) => c.id === conv.id)) {
        conversations.value = [conv, ...conversations.value]
      }
      return conv
    } catch {
      appStore.showError(t('chat.errors.createConversationFailed'))
      return null
    }
  }

  async function selectConversation(id: number): Promise<void> {
    if (currentConversationId.value === id && messages.value.length > 0) return
    currentConversationId.value = id
    messages.value = []
    messagesPrevCursor.value = null
    messagesLoading.value = true
    try {
      // Fetch only the newest page (before_id=0). Earlier history is loaded on
      // demand when the user scrolls to the top (see loadOlderMessages).
      const res = await chatAPI.listMessages(id, { beforeId: 0 })
      messages.value = (res.items ?? []).map(toLiveMessage)
      messagesPrevCursor.value = res.next_cursor ?? null
    } catch {
      appStore.showError(t('chat.errors.loadMessagesFailed'))
    } finally {
      messagesLoading.value = false
    }
  }

  // Fetch one older page and prepend it (deduplicated by server id), preserving
  // chronological order (older messages first). Only runs when an older page
  // remains and another fetch is not already in flight.
  async function loadOlderMessages(): Promise<void> {
    const cursor = messagesPrevCursor.value
    const convId = currentConversationId.value
    if (!cursor || convId == null || messagesLoadingMore.value) return
    messagesLoadingMore.value = true
    try {
      const res = await chatAPI.listMessages(convId, { beforeId: Number(cursor) })
      const older = (res.items ?? []).map(toLiveMessage)
      const existing = new Set(messages.value.map((m) => m.serverId))
      const fresh = older.filter((m) => !existing.has(m.serverId))
      messages.value = [...fresh, ...messages.value]
      messagesPrevCursor.value = res.next_cursor ?? null
    } catch {
      appStore.showError(t('chat.errors.loadMessagesFailed'))
    } finally {
      messagesLoadingMore.value = false
    }
  }

  function startNewConversation(): void {
    currentConversationId.value = null
    messages.value = []
    status.value = 'idle'
  }

  async function renameConversation(id: number, title: string): Promise<void> {
    const clean = title.trim()
    if (!clean) return
    try {
      const updated = await chatAPI.updateConversation(id, { title: clean })
      const idx = conversations.value.findIndex((c) => c.id === id)
      if (idx >= 0) conversations.value[idx] = updated
    } catch {
      appStore.showError(t('chat.errors.renameFailed'))
    }
  }

  async function deleteConversation(id: number): Promise<void> {
    try {
      await chatAPI.deleteConversation(id)
      conversations.value = conversations.value.filter((c) => c.id !== id)
      if (currentConversationId.value === id) {
        startNewConversation()
      }
    } catch {
      appStore.showError(t('chat.errors.deleteFailed'))
    }
  }

  // ==================== Sending / streaming ====================

  // Persist a single message and return the canonical row from the server (if any).
  async function persistOne(
    conversationId: number,
    input: PersistMessageInput
  ): Promise<ChatMessage | null> {
    try {
      const items = await chatAPI.persistMessages(conversationId, [input])
      return items[0] ?? null
    } catch {
      // Persistence failures must not break the live conversation; surface a toast.
      appStore.showError(t('chat.errors.persistFailed'))
      return null
    }
  }

  // Stream an assistant reply for the given live placeholder using the current
  // `messages` (excluding any in-flight assistant with no content). Writes deltas
  // into `target.content` and reports the terminal outcome. Manages the shared
  // abortController and `status` ref. Reusable by sendMessage and regenerate.
  async function streamAssistant(
    model: string,
    target: LiveMessage
  ): Promise<{ content: string; errored: boolean; errorText: string }> {
    status.value = 'streaming'

    // Build the chat-completions message history from persisted/live turns
    // (system messages excluded).
    const history = messages.value
      .filter((m) => m.role === 'user' || (m.role === 'assistant' && m.content))
      .map((m) => ({ role: m.role, content: m.content }))

    abortController.value = new AbortController()
    let acc = ''
    let streamErrored = false
    let errorText = ''

    await chatCompletionStream(
      playground.apiKey,
      { model, messages: history, stream: true },
      {
        onChunk: (_raw, parsed: unknown) => {
          const delta = (parsed as { choices?: Array<{ delta?: { content?: unknown } }> })
            ?.choices?.[0]?.delta
          if (delta && typeof delta.content === 'string' && delta.content) {
            acc += delta.content
            target.content = acc
          }
        },
        onDone: () => {
          target.content = acc
        },
        onError: (err) => {
          streamErrored = true
          errorText = t('playground.errors.httpError', {
            status: err.status || 0,
            message: err.message
          })
        }
      },
      abortController.value.signal
    )
    abortController.value = null

    return { content: acc, errored: streamErrored, errorText }
  }

  async function sendMessage(userText: string): Promise<void> {
    const text = userText.trim()
    if (!text || isStreaming.value) return

    if (!playground.apiKey) {
      appStore.showError(t('playground.errors.noKeySelected'))
      return
    }
    if (!playground.inputs.model) {
      appStore.showError(t('playground.errors.noModelSelected'))
      return
    }

    // Ensure a conversation exists (lazy creation on first send).
    let conversationId = currentConversationId.value
    if (!conversationId) {
      const conv = await createConversation(text)
      if (!conv) return
      conversationId = conv.id
      currentConversationId.value = conv.id
    }

    const model = playground.inputs.model

    // 1) Append + persist the user message.
    const userLive: LiveMessage = {
      id: generateMessageId(),
      role: 'user',
      content: text,
      status: 'complete',
      createdAt: new Date().toISOString()
    }
    messages.value.push(userLive)
    const persistedUser = await persistOne(conversationId, {
      client_message_id: userLive.id,
      role: 'user',
      content: text,
      model,
      status: 'complete'
    })
    // Keep the stable client UUID as `id` (v-for key / client_message_id) and
    // only record the server's numeric id separately.
    if (persistedUser) userLive.serverId = persistedUser.id

    // 2) Append an in-memory assistant placeholder and start streaming.
    const assistant: LiveMessage = {
      id: generateMessageId(),
      role: 'assistant',
      content: '',
      status: 'streaming',
      createdAt: new Date().toISOString()
    }
    messages.value.push(assistant)

    const { errored, errorText } = await streamAssistant(model, assistant)

    // 3) Save the assistant message once, at its terminal state.
    const terminalStatus: ChatMessageStatus = errored ? 'error' : 'complete'
    assistant.status = terminalStatus
    if (errored) {
      assistant.errorMessage = errorText
      status.value = 'error'
      appStore.showError(errorText)
    } else {
      status.value = 'idle'
    }

    const persistedAssistant = await persistOne(conversationId, {
      client_message_id: assistant.id,
      role: 'assistant',
      content: assistant.content,
      model,
      status: terminalStatus
    })
    // Preserve the stable client UUID; store the server id separately.
    if (persistedAssistant) assistant.serverId = persistedAssistant.id
  }

  function stopGeneration(): void {
    if (abortController.value) {
      abortController.value.abort()
      abortController.value = null
    }
    status.value = 'idle'
    const last = messages.value[messages.value.length - 1]
    if (last && last.role === 'assistant' && last.status === 'streaming') {
      last.status = 'complete'
    }
  }

  // Regenerate the last assistant reply: stream a fresh reply from the context up
  // to (and including) the preceding user turn, then atomically replace the old
  // assistant row server-side. "Generate first, replace on success" — if the
  // stream fails or aborts, the original assistant message is restored and the
  // server is left untouched.
  async function regenerate(message: LiveMessage): Promise<void> {
    if (isStreaming.value) return
    // Only the final assistant message is regenerable, and only at rest.
    const last = messages.value[messages.value.length - 1]
    if (
      last !== message ||
      message.role !== 'assistant' ||
      message.status === 'streaming'
    ) {
      return
    }

    const conversationId = currentConversationId.value
    if (conversationId == null) return
    if (!playground.apiKey) {
      appStore.showError(t('playground.errors.noKeySelected'))
      return
    }
    if (!playground.inputs.model) {
      appStore.showError(t('playground.errors.noModelSelected'))
      return
    }
    const model = playground.inputs.model

    // Back up the old assistant so we can restore it on any failure.
    const backup = {
      content: message.content,
      serverId: message.serverId,
      status: message.status
    }
    const restore = () => {
      message.content = backup.content
      message.serverId = backup.serverId
      message.status = backup.status
      status.value = 'idle'
    }

    // Reset the message in place (stable v-for key preserved) and stream a fresh
    // reply. streamAssistant rebuilds history from `messages`, and since this
    // assistant's content is now empty it is excluded from the prompt context.
    message.content = ''
    message.status = 'streaming'
    message.errorMessage = undefined

    const { content, errored, errorText } = await streamAssistant(model, message)

    if (errored) {
      restore()
      appStore.showError(errorText)
      return
    }

    // Atomically replace the old assistant row. Identify the cutoff by server id
    // when known, otherwise by the stable client message id.
    const newClientId = generateMessageId()
    try {
      const replaced = await chatAPI.replaceMessage(conversationId, {
        from_id: backup.serverId,
        from_client_message_id: backup.serverId === undefined ? message.id : undefined,
        message: {
          client_message_id: newClientId,
          role: 'assistant',
          content,
          model,
          status: 'complete'
        }
      })
      message.content = replaced.content
      message.serverId = replaced.id
      message.id = `srv-${replaced.id}`
      message.status = 'complete'
      status.value = 'idle'
    } catch {
      restore()
      appStore.showError(t('chat.errors.regenerateFailed'))
    }
  }

  return {
    // state
    conversations,
    conversationsLoading,
    conversationsLoaded,
    conversationsLoadingMore,
    nextConversationsCursor,
    currentConversationId,
    messages,
    messagesLoading,
    messagesPrevCursor,
    messagesLoadingMore,
    status,
    // getters
    isStreaming,
    // actions
    loadConversations,
    loadMoreConversations,
    createConversation,
    selectConversation,
    loadOlderMessages,
    startNewConversation,
    renameConversation,
    deleteConversation,
    sendMessage,
    regenerate,
    stopGeneration
  }
})
