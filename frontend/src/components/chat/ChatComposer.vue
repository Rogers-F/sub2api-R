<template>
  <div class="flex-shrink-0 pb-5 pt-3.5">
    <div class="mx-auto max-w-[760px] px-6">
      <div
        class="flex items-end gap-2.5 rounded-[18px] border bg-paper-0 px-4 py-2.5 transition-colors focus-within:border-gold-600 dark:bg-ink-800 border-paper-300 dark:border-ink-600"
      >
        <textarea
          ref="taRef"
          v-model="draft"
          rows="1"
          :placeholder="t('chat.composerPlaceholder')"
          :disabled="disabled"
          class="max-h-44 flex-1 resize-none bg-transparent py-1 text-[14.5px] leading-relaxed text-dust-900 placeholder:text-dust-400 focus:outline-none disabled:opacity-60 dark:text-pearl-100 dark:placeholder:text-pearl-500"
          @input="autoGrow"
          @keydown="onKeydown"
        />

        <button
          v-if="streaming"
          type="button"
          class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-coral-600 text-white transition-colors cursor-pointer hover:bg-coral-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-coral-500/60"
          :aria-label="t('chat.actions.stop')"
          :title="t('chat.actions.stop')"
          @click="emit('stop')"
        >
          <svg class="h-[15px] w-[15px]" viewBox="0 0 24 24" fill="currentColor">
            <rect x="6" y="6" width="12" height="12" rx="2" />
          </svg>
        </button>
        <button
          v-else
          type="button"
          class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-gold-400 to-gold-600 text-dust-900 transition-opacity cursor-pointer hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 disabled:cursor-not-allowed disabled:opacity-40"
          :disabled="!canSend"
          :aria-label="t('chat.actions.send')"
          :title="t('chat.actions.send')"
          @click="onSend"
        >
          <svg
            class="h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <path
              stroke-linejoin="round"
              d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"
            />
          </svg>
        </button>
      </div>

      <p class="mt-2.5 text-center text-[11px] text-dust-400 dark:text-pearl-500">
        {{ t('chat.disclaimer') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ streaming: boolean; disabled: boolean }>()
const emit = defineEmits<{
  (e: 'send', text: string): void
  (e: 'stop'): void
}>()

const { t } = useI18n()

const draft = ref('')
const taRef = ref<HTMLTextAreaElement | null>(null)

const canSend = computed(() => draft.value.trim().length > 0 && !props.disabled)

function autoGrow() {
  const el = taRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${el.scrollHeight}px`
}

function onKeydown(e: KeyboardEvent) {
  // Enter sends; Shift+Enter inserts a newline.
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    onSend()
  }
}

function onSend() {
  if (!canSend.value) return
  const text = draft.value.trim()
  draft.value = ''
  nextTick(autoGrow)
  emit('send', text)
}

// Allow parent (e.g. example prompt cards) to prefill the composer.
function setText(text: string) {
  draft.value = text
  nextTick(() => {
    autoGrow()
    taRef.value?.focus()
  })
}

// Re-fit height when the draft is cleared externally.
watch(draft, (v) => {
  if (!v) nextTick(autoGrow)
})

defineExpose({ setText })
</script>
