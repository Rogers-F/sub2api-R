import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useDebounceFn, useStorage } from '@vueuse/core'

import {
  DEFAULT_INPUTS,
  DEFAULT_PARAMETER_ENABLED,
  STORAGE_KEYS,
  STREAM_STATUS
} from '@/constants/playground'
import type {
  ModelOption,
  ParameterEnabled,
  PlaygroundInputs,
  PlaygroundMessage,
  StreamStatus
} from '@/types/playground'
import type { ApiKey, Group } from '@/types'

interface PersistedConfig {
  inputs: PlaygroundInputs
  parameterEnabled: ParameterEnabled
}

interface PersistedMessages {
  messages: PlaygroundMessage[]
}

const MESSAGE_PERSIST_DEBOUNCE_MS = 400

function loadPersistedMessages(): PlaygroundMessage[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEYS.MESSAGES)
    if (!raw) return []
    const parsed = JSON.parse(raw) as PersistedMessages | PlaygroundMessage[]
    if (Array.isArray(parsed)) return parsed
    return parsed?.messages ?? []
  } catch {
    return []
  }
}

export const usePlaygroundStore = defineStore('playground', () => {
  const config = useStorage<PersistedConfig>(STORAGE_KEYS.CONFIG, {
    inputs: { ...DEFAULT_INPUTS },
    parameterEnabled: { ...DEFAULT_PARAMETER_ENABLED }
  })

  // 消息单独走 ref + 防抖持久化（避免每个流式 token 都触发 JSON.stringify+localStorage 写）
  const messages = ref<PlaygroundMessage[]>(loadPersistedMessages())
  const persistMessages = useDebounceFn(() => {
    try {
      localStorage.setItem(STORAGE_KEYS.MESSAGES, JSON.stringify({ messages: messages.value }))
    } catch {
      /* 写入失败（隐私模式 / 配额满）静默忽略 */
    }
  }, MESSAGE_PERSIST_DEBOUNCE_MS)
  watch(messages, () => persistMessages(), { deep: true })

  const status = ref<StreamStatus>(STREAM_STATUS.IDLE)
  const errorMessage = ref<string>('')
  const groups = ref<Group[]>([])
  const apiKeys = ref<ApiKey[]>([])
  const models = ref<ModelOption[]>([])
  const resourcesLoading = ref<boolean>(false)

  const inputs = computed(() => config.value.inputs)
  const parameterEnabled = computed(() => config.value.parameterEnabled)

  const selectedKey = computed<ApiKey | null>(() => {
    const id = config.value.inputs.apiKeyId
    if (id == null) return null
    return apiKeys.value.find((k) => k.id === id) ?? null
  })

  const apiKey = computed<string>(() => selectedKey.value?.key ?? '')

  const currentGroup = computed<Group | null>(() => {
    const name = config.value.inputs.group
    if (!name) return null
    return groups.value.find((g) => g.name === name) ?? null
  })

  const apiKeysForCurrentGroup = computed<ApiKey[]>(() => {
    const group = currentGroup.value
    if (!group) return []
    return apiKeys.value.filter((k) => k.group_id === group.id)
  })

  function setInput<K extends keyof PlaygroundInputs>(key: K, value: PlaygroundInputs[K]) {
    config.value.inputs[key] = value
  }

  function setParameterEnabled<K extends keyof ParameterEnabled>(
    key: K,
    value: ParameterEnabled[K]
  ) {
    config.value.parameterEnabled[key] = value
  }

  function addMessage(m: PlaygroundMessage) {
    messages.value.push(m)
  }

  function patchLastMessage(patch: Partial<PlaygroundMessage>) {
    const list = messages.value
    if (list.length === 0) return
    Object.assign(list[list.length - 1], patch)
  }

  function clearMessages() {
    messages.value = []
  }

  function removeMessage(id: string) {
    messages.value = messages.value.filter((m) => m.id !== id)
  }

  function truncateFromMessage(id: string) {
    const idx = messages.value.findIndex((m) => m.id === id)
    if (idx < 0) return
    messages.value = messages.value.slice(0, idx)
  }

  function setStatus(s: StreamStatus, error = '') {
    status.value = s
    errorMessage.value = error
  }

  return {
    config,
    messages,
    status,
    errorMessage,
    groups,
    apiKeys,
    models,
    resourcesLoading,
    inputs,
    parameterEnabled,
    selectedKey,
    apiKey,
    apiKeysForCurrentGroup,
    currentGroup,
    setInput,
    setParameterEnabled,
    addMessage,
    patchLastMessage,
    clearMessages,
    removeMessage,
    truncateFromMessage,
    setStatus
  }
})
