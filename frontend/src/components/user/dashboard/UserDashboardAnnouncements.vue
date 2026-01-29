<template>
  <div v-if="hasAnnouncement" class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.announcements') }}</h2>
    </div>
    <div class="p-4">
      <!-- Render as iframe if URL -->
      <iframe
        v-if="isUrl"
        :src="homeContent.trim()"
        class="h-48 w-full rounded-lg border-0"
        allowfullscreen
      ></iframe>
      <!-- Render as HTML/Markdown content -->
      <div
        v-else
        class="prose prose-sm max-w-none dark:prose-invert"
        v-html="homeContent"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

const hasAnnouncement = computed(() => !!homeContent.value.trim())

const isUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})
</script>
