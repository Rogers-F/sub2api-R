<template>
  <div
    class="flex h-14 flex-shrink-0 items-center justify-between gap-3 border-b hairline bg-paper-50/80 px-4 backdrop-blur dark:bg-ink-900/70"
  >
    <div class="flex items-center gap-2">
      <!-- Mobile drawer toggle -->
      <button
        type="button"
        class="btn-ghost btn-icon lg:hidden"
        :aria-label="t('chat.openSidebar')"
        @click="emit('toggle-sidebar')"
      >
        <Icon name="menu" size="md" />
      </button>

      <!-- Compact model picker (reuses playground group/key/model resolution) -->
      <div ref="pickerRef" class="relative">
        <button
          type="button"
          :class="[
            'flex items-center gap-2 rounded-[10px] border px-3 py-1.5 text-[13.5px] transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60',
            'border-paper-300 bg-paper-0 text-dust-700 dark:border-ink-600 dark:bg-ink-800 dark:text-pearl-100',
            !hasModel && 'opacity-60'
          ]"
          @click="pickerOpen = !pickerOpen"
        >
          <span
            :class="[
              'h-[7px] w-[7px] rounded-full',
              hasModel ? 'bg-mint-500' : 'bg-pearl-500'
            ]"
          />
          <span class="max-w-[200px] truncate">{{ modelLabel }}</span>
          <Icon name="chevronDown" size="sm" />
        </button>

        <transition name="chat-pop">
          <div
            v-if="pickerOpen"
            class="dropdown absolute left-0 top-full z-40 mt-2 w-72 p-3"
          >
            <KeyGroupModelPicker />
          </div>
        </transition>
      </div>
    </div>

    <div class="flex items-center gap-2.5">
      <button
        type="button"
        class="flex items-center gap-1.5 rounded-[10px] border px-3 py-1.5 text-[13px] font-semibold transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 border-paper-300 bg-paper-0 text-dust-600 hover:border-gold-400 hover:text-gold-700 dark:border-ink-600 dark:bg-ink-800 dark:text-pearl-200 dark:hover:border-gold-600 dark:hover:text-gold-200"
        @click="goDeveloperCenter"
      >
        <svg
          class="h-[15px] w-[15px]"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M16 18l6-6-6-6M8 6l-6 6 6 6" />
        </svg>
        <span class="hidden sm:inline">{{ t('nav.devCenter') }}</span>
      </button>

      <button
        type="button"
        class="flex h-[34px] w-[34px] items-center justify-center rounded-[9px] border transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 border-paper-300 bg-paper-0 text-dust-500 hover:text-gold-700 dark:border-ink-600 dark:bg-ink-800 dark:text-pearl-300 dark:hover:text-gold-200"
        :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
        :title="isDark ? t('nav.lightMode') : t('nav.darkMode')"
        @click="toggleTheme"
      >
        <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import KeyGroupModelPicker from '@/components/playground/KeyGroupModelPicker.vue'
import { usePlaygroundStore } from '@/stores/playground'

const emit = defineEmits<{ (e: 'toggle-sidebar'): void }>()

const router = useRouter()
const { t } = useI18n()
const playground = usePlaygroundStore()

const pickerOpen = ref(false)
const pickerRef = ref<HTMLElement | null>(null)
const isDark = ref(document.documentElement.classList.contains('dark'))

const hasModel = computed(() => !!playground.inputs.model)
const modelLabel = computed(() => {
  if (!playground.inputs.model) return t('chat.noModel')
  const opt = playground.models.find((m) => m.value === playground.inputs.model)
  return opt?.label || playground.inputs.model
})

function goDeveloperCenter() {
  void router.push('/dashboard')
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function handleClickOutside(e: MouseEvent) {
  if (pickerRef.value && !pickerRef.value.contains(e.target as Node)) {
    pickerOpen.value = false
  }
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))
</script>

<style scoped>
.chat-pop-enter-active,
.chat-pop-leave-active {
  transition: all 0.18s ease;
}
.chat-pop-enter-from,
.chat-pop-leave-to {
  opacity: 0;
  transform: scale(0.97) translateY(-4px);
}
</style>
