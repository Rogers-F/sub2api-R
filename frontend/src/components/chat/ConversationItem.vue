<template>
  <div
    :class="[
      'group relative rounded-xl border-l-2 transition-colors',
      active
        ? 'border-gold-500 bg-paper-0 shadow-sm dark:bg-ink-700'
        : 'border-transparent hover:bg-paper-200 dark:hover:bg-ink-800'
    ]"
  >
    <button
      type="button"
      class="flex w-full flex-col gap-0.5 rounded-xl px-3 py-2.5 text-left cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60"
      @click="emit('select', conversation.id)"
    >
      <span
        :class="[
          'truncate pr-6 text-[13.5px]',
          active
            ? 'font-semibold text-dust-900 dark:text-pearl-50'
            : 'text-dust-700 dark:text-pearl-100'
        ]"
      >
        {{ conversation.title || t('chat.untitled') }}
      </span>
      <span class="text-[11.5px] text-dust-400 dark:text-pearl-500">
        {{ relativeTime }}
      </span>
    </button>

    <!-- Hover-reveal delete (sibling button, not nested inside the select button) -->
    <button
      type="button"
      :aria-label="t('chat.actions.delete')"
      :title="t('chat.actions.delete')"
      class="absolute right-2 top-2.5 rounded-md p-0.5 text-dust-400 opacity-0 transition-colors cursor-pointer hover:text-coral-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-coral-500/60 group-hover:opacity-100 dark:text-pearl-400"
      @click.stop="emit('delete', conversation.id)"
    >
      <Icon name="trash" size="sm" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import { formatRelativeTime } from '@/utils/format'
import type { Conversation } from '@/types/chat'

const props = defineProps<{ conversation: Conversation; active: boolean }>()
const emit = defineEmits<{
  (e: 'select', id: number): void
  (e: 'delete', id: number): void
}>()

const { t } = useI18n()

const relativeTime = computed(() =>
  formatRelativeTime(props.conversation.updated_at || props.conversation.created_at)
)
</script>
