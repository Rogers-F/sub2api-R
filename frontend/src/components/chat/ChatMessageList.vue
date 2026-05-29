<template>
  <div ref="scrollRef" class="flex-1 overflow-y-auto py-7" @scroll="onScroll">
    <div class="mx-auto flex max-w-[760px] flex-col gap-[22px] px-6">
      <ChatMessageItem v-for="msg in messages" :key="msg.id" :message="msg" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'

import ChatMessageItem from './ChatMessageItem.vue'
import type { LiveMessage } from '@/stores/chat'

const props = defineProps<{ messages: LiveMessage[]; streaming: boolean }>()

const scrollRef = ref<HTMLElement | null>(null)
// Auto-scroll to bottom unless the user has scrolled up.
const stickToBottom = ref(true)
const PIN_THRESHOLD_PX = 48

function onScroll() {
  const el = scrollRef.value
  if (!el) return
  stickToBottom.value = el.scrollTop + el.clientHeight >= el.scrollHeight - PIN_THRESHOLD_PX
}

// A cheap value-comparable signature (count + last message length) so the watch
// fires on new messages and on streaming growth without deep-comparing the array.
watch(
  () => {
    const last = props.messages[props.messages.length - 1]
    return `${props.messages.length}:${last ? last.content.length : 0}`
  },
  () => {
    if (!stickToBottom.value) return
    nextTick(() => {
      const el = scrollRef.value
      if (el) el.scrollTop = el.scrollHeight
    })
  }
)
</script>
