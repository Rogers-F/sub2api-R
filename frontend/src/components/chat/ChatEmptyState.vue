<template>
  <div class="flex flex-1 flex-col items-center justify-center px-6 py-6">
    <div
      class="mb-5 flex h-[60px] w-[60px] items-center justify-center rounded-[18px] bg-gradient-to-br from-gold-400 to-gold-600 text-[26px] font-extrabold text-dust-900"
    >
      {{ t('chat.brandInitial') }}
    </div>
    <h2 class="mb-2 text-[23px] font-bold text-dust-900 dark:text-pearl-50">
      {{ t('chat.empty.title') }}
    </h2>
    <p class="mb-7 text-sm text-dust-400 dark:text-pearl-400">
      {{ t('chat.empty.subtitle') }}
    </p>

    <div class="grid w-full max-w-[600px] grid-cols-1 gap-3 sm:grid-cols-2">
      <button
        v-for="(ex, idx) in examples"
        :key="idx"
        type="button"
        class="rounded-2xl border p-4 text-left transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 border-paper-300 bg-paper-0 hover:border-gold-400 dark:border-ink-700 dark:bg-ink-800 dark:hover:border-gold-600 dark:hover:bg-ink-700"
        @click="emit('pick', ex.prompt)"
      >
        <div class="mb-1 text-[13.5px] font-semibold text-dust-800 dark:text-pearl-100">
          {{ ex.title }}
        </div>
        <div class="text-xs text-dust-400 dark:text-pearl-500">{{ ex.prompt }}</div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const emit = defineEmits<{ (e: 'pick', prompt: string): void }>()

const { t } = useI18n()

// Four guided example prompts; clicking one fills the composer.
const examples = computed(() => [
  { title: t('chat.empty.examples.code.title'), prompt: t('chat.empty.examples.code.prompt') },
  { title: t('chat.empty.examples.polish.title'), prompt: t('chat.empty.examples.polish.prompt') },
  {
    title: t('chat.empty.examples.explain.title'),
    prompt: t('chat.empty.examples.explain.prompt')
  },
  { title: t('chat.empty.examples.plan.title'), prompt: t('chat.empty.examples.plan.prompt') }
])
</script>
