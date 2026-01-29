<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.quickStart') }}</h2>
    </div>
    <div class="space-y-3 p-4">
      <!-- Documentation Link -->
      <a
        v-if="docUrl"
        :href="docUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="group flex w-full items-center gap-4 rounded-xl bg-gray-50 p-4 text-left transition-all duration-200 hover:bg-gray-100 dark:bg-dark-800/50 dark:hover:bg-dark-800"
      >
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-blue-100 transition-transform group-hover:scale-105 dark:bg-blue-900/30">
          <Icon name="document" size="lg" class="text-blue-600 dark:text-blue-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('dashboard.documentation') }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.viewDocumentation') }}</p>
        </div>
        <Icon
          name="arrowUpRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-blue-500 dark:text-dark-500"
        />
      </a>

      <!-- API Base URL -->
      <div
        v-if="apiBaseUrl"
        class="group flex w-full items-center gap-4 rounded-xl bg-gray-50 p-4 text-left transition-all duration-200 dark:bg-dark-800/50"
      >
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30">
          <Icon name="link" size="lg" class="text-emerald-600 dark:text-emerald-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('dashboard.apiEndpoint') }}</p>
          <p class="text-xs font-mono text-gray-500 dark:text-dark-400 truncate">{{ apiBaseUrl }}</p>
        </div>
        <button
          @click="copyApiUrl"
          class="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-300"
          :title="t('common.copy')"
        >
          <Icon :name="copied ? 'check' : 'copy'" size="sm" />
        </button>
      </div>

      <!-- No Data -->
      <div v-if="!docUrl && !apiBaseUrl" class="py-4 text-center text-sm text-gray-400">
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
