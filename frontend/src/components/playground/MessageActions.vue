<template>
  <div class="absolute right-2 top-2 hidden gap-1 group-hover:flex">
    <button
      class="rounded-md p-1.5 text-dust-400 hover-tint hover:text-primary-fg"
      :title="t('playground.chat.copyMessage')"
      @click="onCopy"
    >
      <Icon name="copy" size="sm" />
    </button>
    <button
      v-if="canRetry"
      class="rounded-md p-1.5 text-dust-400 hover-tint hover:text-primary-fg disabled:opacity-40"
      :title="t('playground.chat.retryMessage')"
      :disabled="isStreaming"
      @click="emit('retry')"
    >
      <Icon name="refresh" size="sm" />
    </button>
    <button
      class="rounded-md p-1.5 text-dust-400 hover-tint hover:text-coral-600 disabled:opacity-40"
      :title="t('playground.chat.deleteMessage')"
      :disabled="isStreaming"
      @click="emit('delete')"
    >
      <Icon name="trash" size="sm" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { MESSAGE_ROLES, STREAM_STATUS } from '@/constants/playground'
import { usePlaygroundStore } from '@/stores/playground'
import { getTextContent } from '@/utils/playground/messageHelpers'
import type { PlaygroundMessage } from '@/types/playground'

const props = defineProps<{ message: PlaygroundMessage }>()
const emit = defineEmits<{ (e: 'delete'): void; (e: 'retry'): void }>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const store = usePlaygroundStore()

const isStreaming = computed(() => store.status === STREAM_STATUS.STREAMING)
const canRetry = computed(() => props.message.role === MESSAGE_ROLES.USER)

function onCopy() {
  copyToClipboard(getTextContent(props.message))
}
</script>
