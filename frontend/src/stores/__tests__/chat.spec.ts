import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// --- Mocks ---

const mockListConversations = vi.fn()
const mockCreateConversation = vi.fn()
const mockListMessages = vi.fn()
const mockPersistMessages = vi.fn()
const mockDeleteConversation = vi.fn()
const mockUpdateConversation = vi.fn()

vi.mock('@/api/chat', () => ({
  chatAPI: {
    listConversations: (...a: any[]) => mockListConversations(...a),
    createConversation: (...a: any[]) => mockCreateConversation(...a),
    getConversation: vi.fn(),
    updateConversation: (...a: any[]) => mockUpdateConversation(...a),
    deleteConversation: (...a: any[]) => mockDeleteConversation(...a),
    listMessages: (...a: any[]) => mockListMessages(...a),
    persistMessages: (...a: any[]) => mockPersistMessages(...a)
  }
}))

// Drive the streaming handlers deterministically.
const mockChatCompletionStream = vi.fn()
vi.mock('@/api/playground', () => ({
  chatCompletionStream: (...a: any[]) => mockChatCompletionStream(...a)
}))

// i18n: identity translator so assertions stay simple.
vi.mock('@/i18n', () => ({
  i18n: { global: { t: (key: string) => key } }
}))

import { useChatStore } from '@/stores/chat'
import { usePlaygroundStore } from '@/stores/playground'

function primePlayground() {
  const pg = usePlaygroundStore()
  // selectedKey derives from apiKeyId; seed an active key + model.
  pg.apiKeys = [
    {
      id: 1,
      user_id: 1,
      key: 'sk-test-key',
      name: 'k',
      group_id: 1,
      status: 'active',
      ip_whitelist: [],
      ip_blacklist: [],
      last_used_at: null,
      quota: 0,
      quota_used: 0,
      expires_at: null,
      created_at: '',
      updated_at: '',
      rate_limit_5h: 0,
      rate_limit_1d: 0,
      rate_limit_7d: 0,
      usage_5h: 0,
      usage_1d: 0,
      usage_7d: 0,
      window_5h_start: null,
      window_1d_start: null,
      window_7d_start: null,
      reset_5h_at: null,
      reset_1d_at: null,
      reset_7d_at: null
    } as any
  ]
  pg.setInput('apiKeyId', 1)
  pg.setInput('model', 'test-model')
  return pg
}

describe('useChatStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    localStorage.clear()
  })

  describe('loadConversations', () => {
    it('populates the conversation list', async () => {
      mockListConversations.mockResolvedValue({
        items: [{ id: 1, title: 'A', model: 'm', status: 'active', created_at: '', updated_at: '' }],
        next_cursor: null
      })
      const chat = useChatStore()
      await chat.loadConversations()
      expect(chat.conversations).toHaveLength(1)
      expect(chat.conversationsLoaded).toBe(true)
      expect(chat.nextConversationsCursor).toBeNull()
    })

    it('appends the next page via loadMoreConversations and dedupes by id', async () => {
      mockListConversations.mockResolvedValueOnce({
        items: [{ id: 1, title: 'A', model: 'm', status: 'active', created_at: '', updated_at: '' }],
        next_cursor: 'cursor-2'
      })
      const chat = useChatStore()
      await chat.loadConversations()
      expect(chat.nextConversationsCursor).toBe('cursor-2')

      mockListConversations.mockResolvedValueOnce({
        items: [
          // Duplicate id 1 must be ignored; id 2 appended.
          { id: 1, title: 'A', model: 'm', status: 'active', created_at: '', updated_at: '' },
          { id: 2, title: 'B', model: 'm', status: 'active', created_at: '', updated_at: '' }
        ],
        next_cursor: null
      })
      await chat.loadMoreConversations()
      expect(chat.conversations.map((c) => c.id)).toEqual([1, 2])
      expect(chat.nextConversationsCursor).toBeNull()
    })
  })

  describe('selectConversation', () => {
    it('loads persisted messages for the conversation', async () => {
      mockListMessages.mockResolvedValue({
        items: [
          {
            id: 1,
            conversation_id: 1,
            role: 'user',
            content: 'hello',
            model: 'm',
            status: 'complete',
            created_at: ''
          }
        ],
        next_cursor: null
      })
      const chat = useChatStore()
      await chat.selectConversation(1)
      expect(chat.currentConversationId).toBe(1)
      expect(chat.messages).toHaveLength(1)
      expect(chat.messages[0].content).toBe('hello')
      // Live id stays a stable string; the numeric server id is kept separately.
      expect(typeof chat.messages[0].id).toBe('string')
      expect(chat.messages[0].serverId).toBe(1)
    })

    it('loads the full history by following next_cursor across pages', async () => {
      mockListMessages
        .mockResolvedValueOnce({
          items: [
            { id: 1, conversation_id: 1, role: 'user', content: 'first', model: 'm', status: 'complete', created_at: '' }
          ],
          next_cursor: 'page-2'
        })
        .mockResolvedValueOnce({
          items: [
            { id: 2, conversation_id: 1, role: 'assistant', content: 'latest', model: 'm', status: 'complete', created_at: '' }
          ],
          next_cursor: null
        })
      const chat = useChatStore()
      await chat.selectConversation(1)
      expect(mockListMessages).toHaveBeenCalledTimes(2)
      expect(chat.messages.map((m) => m.content)).toEqual(['first', 'latest'])
    })
  })

  describe('sendMessage', () => {
    it('lazily creates a conversation, streams, and persists both turns', async () => {
      primePlayground()
      mockCreateConversation.mockResolvedValue({
        id: 101,
        title: 'hi there',
        model: 'test-model',
        status: 'active',
        created_at: '',
        updated_at: ''
      })
      // Server echoes a persisted row with a numeric id.
      mockPersistMessages.mockResolvedValue([
        { id: 9001, conversation_id: 101, role: 'user', content: '', model: 'test-model', status: 'complete', created_at: '' }
      ])
      mockChatCompletionStream.mockImplementation(async (_key, _payload, handlers) => {
        handlers.onChunk('', { choices: [{ delta: { content: 'Hello ' } }] })
        handlers.onChunk('', { choices: [{ delta: { content: 'world' } }] })
        handlers.onDone()
      })

      const chat = useChatStore()
      await chat.sendMessage('hi there')

      // Conversation created on first send
      expect(mockCreateConversation).toHaveBeenCalledTimes(1)
      expect(chat.currentConversationId).toBe(101)

      // User + assistant accumulated in memory
      expect(chat.messages).toHaveLength(2)
      expect(chat.messages[0].role).toBe('user')
      expect(chat.messages[1].role).toBe('assistant')
      expect(chat.messages[1].content).toBe('Hello world')
      expect(chat.messages[1].status).toBe('complete')

      // Live ids stay stable string UUIDs (v-for key / client_message_id); the
      // server's numeric id is recorded separately and never overwrites `id`.
      expect(typeof chat.messages[0].id).toBe('string')
      expect(chat.messages[0].serverId).toBe(9001)

      // Persisted once for the user turn and once for the assistant turn
      expect(mockPersistMessages).toHaveBeenCalledTimes(2)
      expect(chat.status).toBe('idle')
    })

    it('marks the assistant message as error when the stream errors', async () => {
      primePlayground()
      mockCreateConversation.mockResolvedValue({
        id: 202,
        title: 't',
        model: 'test-model',
        status: 'active',
        created_at: '',
        updated_at: ''
      })
      mockPersistMessages.mockResolvedValue([])
      mockChatCompletionStream.mockImplementation(async (_key, _payload, handlers) => {
        handlers.onError({ status: 500, message: 'boom' })
      })

      const chat = useChatStore()
      await chat.sendMessage('trigger error')

      const assistant = chat.messages[chat.messages.length - 1]
      expect(assistant.status).toBe('error')
      expect(chat.status).toBe('error')
    })

    it('does not send when no model is selected', async () => {
      const chat = useChatStore()
      await chat.sendMessage('no model')
      expect(mockChatCompletionStream).not.toHaveBeenCalled()
      expect(chat.messages).toHaveLength(0)
    })
  })

  describe('deleteConversation', () => {
    it('removes the conversation and resets the active selection', async () => {
      mockDeleteConversation.mockResolvedValue(undefined)
      const chat = useChatStore()
      chat.conversations = [
        { id: 1, title: 'A', model: 'm', status: 'active', created_at: '', updated_at: '' }
      ]
      chat.currentConversationId = 1
      await chat.deleteConversation(1)
      expect(chat.conversations).toHaveLength(0)
      expect(chat.currentConversationId).toBeNull()
    })
  })
})
