<template>
  <div ref="scrollRef" class="flex-1 overflow-y-auto py-7" @scroll="onScroll">
    <div class="mx-auto flex max-w-[760px] flex-col gap-[22px] px-6">
      <div v-if="loadingOlder" class="py-2 text-center text-xs text-dust-400 dark:text-pearl-500">
        {{ t('chat.loadingEarlier') }}
      </div>
      <ChatMessageItem
        v-for="(msg, index) in messages"
        :key="msg.id"
        :message="msg"
        :is-last="index === messages.length - 1"
        @regenerate="emit('regenerate', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import ChatMessageItem from './ChatMessageItem.vue'
import type { LiveMessage } from '@/stores/chat'

const props = defineProps<{
  messages: LiveMessage[]
  streaming: boolean
  hasOlder?: boolean
  loadingOlder?: boolean
}>()
const emit = defineEmits<{
  (e: 'load-older'): void
  (e: 'regenerate', message: LiveMessage): void
}>()

const { t } = useI18n()

const scrollRef = ref<HTMLElement | null>(null)
// Auto-scroll to bottom unless the user has scrolled up.
const stickToBottom = ref(true)
const PIN_THRESHOLD_PX = 48
const TOP_THRESHOLD_PX = 64

// While a "load older" prepend is in flight we anchor the scroll position to the
// previously top-most content instead of pinning to the bottom, so the view does
// not jump when older messages are inserted at the head.
const prepending = ref(false)
const prevScrollHeight = ref(0)

function onScroll() {
  const el = scrollRef.value
  if (!el) return
  stickToBottom.value = el.scrollTop + el.clientHeight >= el.scrollHeight - PIN_THRESHOLD_PX
  if (el.scrollTop < TOP_THRESHOLD_PX && props.hasOlder && !props.loadingOlder && !prepending.value) {
    prepending.value = true
    prevScrollHeight.value = el.scrollHeight
    emit('load-older')
  }
}

// A cheap value-comparable signature (count + last message length) so the watch
// fires on new messages and on streaming growth without deep-comparing the array.
watch(
  () => {
    const last = props.messages[props.messages.length - 1]
    return `${props.messages.length}:${last ? last.content.length : 0}`
  },
  () => {
    if (prepending.value) {
      // Older page prepended: keep the prior content under the same viewport by
      // compensating for the added height at the top. Clear the flag here (the
      // content list changed) so it cannot block subsequent loads.
      prepending.value = false
      const oldHeight = prevScrollHeight.value
      nextTick(() => {
        const el = scrollRef.value
        if (el) el.scrollTop += el.scrollHeight - oldHeight
      })
      return
    }
    if (!stickToBottom.value) return
    nextTick(() => {
      const el = scrollRef.value
      if (el) el.scrollTop = el.scrollHeight
    })
  }
)

// Safety net: if a load-older round trip ends without changing the message list
// (e.g. an empty/fully-deduplicated page so the signature watch never fires),
// clear the prepend flag so future top-scroll loads are not blocked.
watch(
  () => props.loadingOlder,
  (loading) => {
    if (!loading) prepending.value = false
  }
)
</script>
