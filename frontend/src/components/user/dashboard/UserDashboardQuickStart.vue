<template>
  <div class="card">
    <div class="border-b hairline px-6 py-4">
      <h2 class="text-lg font-semibold text-dust-900 dark:text-white">{{ t('dashboard.quickStart') }}</h2>
    </div>
    <div class="space-y-3 p-4">
      <!-- Documentation Link -->
      <a
        v-if="docUrl"
        :href="docUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="group flex w-full items-center gap-4 rounded-xl bg-paper-50 p-4 text-left transition-all duration-200 hover:bg-paper-100 dark:bg-ink-800/50 dark:hover:bg-ink-800"
      >
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-blue-100 transition-transform group-hover:scale-105 dark:bg-blue-900/30">
          <Icon name="document" size="lg" class="text-blue-600 dark:text-blue-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-dust-900 dark:text-white">{{ t('dashboard.documentation') }}</p>
          <p class="text-xs text-secondary-fg">{{ t('dashboard.viewDocumentation') }}</p>
        </div>
        <Icon
          name="externalLink"
          size="md"
          class="text-dust-400 transition-colors group-hover:text-blue-500 dark:text-pearl-400"
        />
      </a>

      <!-- API Base URL -->
      <div
        v-if="apiBaseUrl"
        class="group flex w-full items-center gap-4 rounded-xl bg-paper-50 p-4 text-left transition-all duration-200 dark:bg-ink-800/50"
      >
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-mint-500/15 dark:bg-mint-500/20">
          <Icon name="link" size="lg" class="text-mint-600 dark:text-mint-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-dust-900 dark:text-white">{{ t('dashboard.apiEndpoint') }}</p>
          <p class="text-xs font-mono text-secondary-fg truncate">{{ apiBaseUrl }}</p>
        </div>
        <button
          @click="copyApiUrl"
          class="flex h-8 w-8 items-center justify-center rounded-lg text-dust-400 transition-colors hover:bg-paper-200 hover:text-dust-600 dark:hover:bg-white/[0.04] dark:hover:text-pearl-100"
          :title="t('common.copy')"
        >
          <Icon :name="copied ? 'check' : 'copy'" size="sm" />
        </button>
      </div>

      <!-- No Data -->
      <div v-if="!docUrl && !apiBaseUrl" class="py-4 text-center text-sm text-dust-400">
        {{ t('dashboard.noQuickStartItems') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const apiBaseUrl = computed(() => appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || '')

const copied = ref(false)

const copyApiUrl = async () => {
  if (!apiBaseUrl.value) return
  try {
    await navigator.clipboard.writeText(apiBaseUrl.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (e) {
    console.error('Failed to copy:', e)
  }
}
</script>
