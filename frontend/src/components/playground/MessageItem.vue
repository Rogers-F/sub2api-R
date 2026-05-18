<template>
  <div
    :class="[
      'panel group relative rounded-2xl p-4 hairline',
      message.role === MESSAGE_ROLES.USER && 'bg-paper-100 dark:bg-ink-700',
      message.role === MESSAGE_ROLES.ASSISTANT &&
        'bg-paper-50 dark:bg-ink-800 border-l-2 border-gold-500',
      message.role === MESSAGE_ROLES.SYSTEM && 'bg-paper-50 dark:bg-ink-900'
    ]"
  >
    <div class="mb-2 flex items-center gap-2 text-xs">
      <span class="font-semibold">{{ roleLabel }}</span>
      <span
        v-if="message.status === MESSAGE_STATUS.LOADING || message.status === MESSAGE_STATUS.INCOMPLETE"
        class="xs-pulse-dot text-mint-700"
      >
        {{ t('playground.chat.streaming') }}
      </span>
      <span
        v-else-if="message.status === MESSAGE_STATUS.ERROR"
        class="badge badge-danger text-xs"
      >
        {{ message.errorMessage || 'error' }}
      </span>
    </div>

    <details
      v-if="message.reasoningContent"
      class="mb-3 rounded-lg bg-purple-500/5 p-2 text-xs"
      :open="message.isReasoningExpanded ?? true"
    >
      <summary class="cursor-pointer text-secondary-fg select-none">
        {{ t('playground.chat.reasoning') }}
      </summary>
      <div
        class="prose prose-sm dark:prose-invert mt-2 max-w-none text-secondary-fg"
        v-html="renderedReasoning"
      />
    </details>

    <!-- 多模态：纯字符串走 markdown；array 形式拆分文本/图片 -->
    <template v-if="typeof message.content === 'string'">
      <div
        class="prose prose-sm dark:prose-invert max-w-none break-words"
        v-html="renderedContent"
      />
    </template>
    <template v-else>
      <template v-for="(part, idx) in message.content" :key="idx">
        <div
          v-if="part.type === 'text'"
          class="prose prose-sm dark:prose-invert max-w-none break-words"
          v-html="renderText(part.text)"
        />
        <img
          v-else-if="part.type === 'image_url'"
          :src="part.image_url.url"
          alt="user image"
          class="my-2 max-h-48 rounded-lg"
        />
      </template>
    </template>

    <MessageActions
      :message="message"
      @delete="emit('delete', message.id)"
      @retry="emit('retry', message.id)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import MessageActions from './MessageActions.vue'
import { MESSAGE_ROLES, MESSAGE_STATUS } from '@/constants/playground'
import { renderMarkdown } from '@/utils/playground/markdownRenderer'
import type { PlaygroundMessage } from '@/types/playground'

const props = defineProps<{ message: PlaygroundMessage }>()
const emit = defineEmits<{
  (e: 'delete', id: string): void
  (e: 'retry', id: string): void
}>()
const { t } = useI18n()

const roleLabel = computed(() => {
  if (props.message.role === MESSAGE_ROLES.USER) return t('playground.chat.roleUser')
  if (props.message.role === MESSAGE_ROLES.ASSISTANT) return t('playground.chat.roleAssistant')
  return t('playground.chat.roleSystem')
})

const renderedContent = computed(() =>
  typeof props.message.content === 'string' ? renderMarkdown(props.message.content) : ''
)
const renderedReasoning = computed(() => renderMarkdown(props.message.reasoningContent ?? ''))

function renderText(text: string) {
  return renderMarkdown(text)
}
</script>
