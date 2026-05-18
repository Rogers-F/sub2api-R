<template>
  <div ref="scrollRef" class="flex-1 overflow-y-auto px-4 py-6 space-y-4" @scroll="onScroll">
    <div
      v-if="store.messages.length === 0"
      class="flex h-full items-center justify-center"
    >
      <p class="text-secondary-fg text-sm">{{ t('playground.chat.empty') }}</p>
    </div>

    <MessageItem
      v-for="msg in store.messages"
      :key="msg.id"
      :message="msg"
      @delete="onDelete"
      @retry="onRetry"
    />
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import MessageItem from './MessageItem.vue'
import { MESSAGE_ROLES } from '@/constants/playground'
import { usePlaygroundStore } from '@/stores/playground'
import { useChatStream } from '@/composables/playground/useChatStream'
import { getTextContent } from '@/utils/playground/messageHelpers'

const store = usePlaygroundStore()
const chat = useChatStream()
const { t } = useI18n()

const scrollRef = ref<HTMLElement | null>(null)
// 用户手动向上滚动后暂停自动滚底，回到底部时恢复
const stickToBottom = ref(true)
const PIN_THRESHOLD_PX = 32

function onScroll() {
  const el = scrollRef.value
  if (!el) return
  stickToBottom.value = el.scrollTop + el.clientHeight >= el.scrollHeight - PIN_THRESHOLD_PX
}

function lastSignature(): [number, number, number] {
  const last = store.messages[store.messages.length - 1]
  if (!last) return [0, 0, 0]
  const contentLen = typeof last.content === 'string' ? last.content.length : last.content.length
  const reasoningLen = last.reasoningContent?.length ?? 0
  return [store.messages.length, contentLen, reasoningLen]
}

watch(lastSignature, () => {
  if (!stickToBottom.value) return
  nextTick(() => {
    const el = scrollRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
})

function onDelete(id: string) {
  store.removeMessage(id)
}

function onRetry(id: string) {
  const msg = store.messages.find((m) => m.id === id)
  if (!msg || msg.role !== MESSAGE_ROLES.USER) return
  const text = getTextContent(msg)
  store.truncateFromMessage(id)
  void chat.sendMessage(text)
}
</script>
