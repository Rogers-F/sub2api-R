<template>
  <div class="flex gap-3">
    <!-- Avatar -->
    <div
      :class="[
        'flex h-[30px] w-[30px] flex-shrink-0 items-center justify-center rounded-lg text-xs font-bold',
        isUser
          ? 'bg-aurora-600 text-white'
          : 'bg-gradient-to-br from-gold-400 to-gold-600 text-dust-900'
      ]"
    >
      {{ avatarText }}
    </div>

    <div class="min-w-0 flex-1">
      <div class="mb-1.5 text-xs font-semibold text-dust-400 dark:text-pearl-400">
        {{ roleLabel }}
      </div>

      <!-- User: aurora-tinted bubble -->
      <div
        v-if="isUser"
        class="rounded-2xl border border-aurora-500/[0.16] bg-aurora-500/[0.08] px-4 py-3 text-[14.5px] leading-relaxed text-dust-900 dark:border-aurora-400/[0.18] dark:bg-aurora-400/10 dark:text-pearl-50"
      >
        <p class="whitespace-pre-wrap break-words">{{ message.content }}</p>
      </div>

      <!-- Assistant: rendered Markdown + optional streaming cursor -->
      <div v-else>
        <div
          v-if="message.status === 'error'"
          class="rounded-xl border border-coral-500/30 bg-coral-500/10 px-4 py-3 text-sm text-coral-600 dark:text-coral-400"
        >
          {{ message.errorMessage || t('chat.errors.generic') }}
        </div>
        <div
          v-else
          class="chat-markdown prose prose-sm max-w-none break-words text-[14.5px] leading-[1.72] text-dust-800 dark:prose-invert dark:text-pearl-100"
        >
          <!-- While streaming, render plain text (markdown is parsed once at the
               terminal state) to avoid re-parsing the whole reply on every token. -->
          <span v-if="message.status === 'streaming'" class="whitespace-pre-wrap break-words">{{
            message.content
          }}</span>
          <span v-else v-html="renderedContent" />
          <span
            v-if="message.status === 'streaming'"
            class="chat-cursor"
            aria-hidden="true"
          />
        </div>

        <!-- Actions -->
        <div v-if="showActions" class="mt-2.5 flex gap-3.5">
          <button
            type="button"
            class="flex items-center gap-1.5 text-xs text-dust-400 transition-colors cursor-pointer hover:text-gold-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 dark:text-pearl-400 dark:hover:text-gold-200"
            @click="onCopy"
          >
            <Icon name="copy" size="xs" />
            {{ t('chat.actions.copy') }}
          </button>
          <button
            v-if="showRegenerate"
            type="button"
            class="flex items-center gap-1.5 text-xs text-dust-400 transition-colors cursor-pointer hover:text-gold-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 dark:text-pearl-400 dark:hover:text-gold-200"
            @click="onRegenerate"
          >
            <Icon name="refresh" size="xs" />
            {{ t('chat.actions.regenerate') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { renderMarkdown } from '@/utils/playground/markdownRenderer'
import { useAuthStore } from '@/stores/auth'
import type { LiveMessage } from '@/stores/chat'

const props = withDefaults(defineProps<{ message: LiveMessage; isLast?: boolean }>(), {
  isLast: false
})
const emit = defineEmits<{ (e: 'regenerate', message: LiveMessage): void }>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const authStore = useAuthStore()

const isUser = computed(() => props.message.role === 'user')
const roleLabel = computed(() => (isUser.value ? t('chat.roleUser') : t('chat.assistantName')))

const userInitial = computed(() => {
  const u = authStore.user
  const name = u?.username || u?.email?.split('@')[0] || '?'
  return name.slice(0, 1).toUpperCase()
})
const avatarText = computed(() => (isUser.value ? userInitial.value : t('chat.brandInitial')))

// Parse markdown only at the terminal state; during streaming the template
// shows plain text, so we avoid an O(n^2) re-parse of the growing reply.
const renderedContent = computed(() =>
  props.message.status === 'streaming' ? '' : renderMarkdown(props.message.content)
)

// Show the copy action once there is content and we are not mid-stream on this message.
const showActions = computed(
  () => props.message.status !== 'streaming' && props.message.content.length > 0
)

// Regenerate is only offered on the final assistant reply, once it is at rest
// with content (so the user can re-roll the last turn).
const showRegenerate = computed(
  () =>
    !isUser.value &&
    props.isLast &&
    props.message.status !== 'streaming' &&
    props.message.content.length > 0
)

function onCopy() {
  void copyToClipboard(props.message.content)
}

function onRegenerate() {
  emit('regenerate', props.message)
}
</script>

<style scoped>
.chat-cursor {
  display: inline-block;
  width: 8px;
  height: 17px;
  margin-left: 2px;
  vertical-align: -3px;
  animation: chat-blink 1s steps(2) infinite;
  @apply bg-gold-400;
}

@keyframes chat-blink {
  0%,
  50% {
    opacity: 1;
  }
  50.01%,
  100% {
    opacity: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-cursor {
    animation: none;
  }
}
</style>
