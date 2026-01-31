<template>
  <div class="card">
    <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div class="flex items-center gap-2">
        <Icon name="megaphone" size="md" class="text-primary-500" />
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.announcements') }}</h2>
      </div>
      <span
        v-if="announcements.length > 0"
        class="rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
      >
        {{ announcements.length }}
      </span>
    </div>
    <div class="p-4">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <LoadingSpinner size="md" />
      </div>

      <!-- Announcements List -->
      <div v-else-if="announcements.length > 0" class="space-y-4">
        <div
          v-for="announcement in announcements.slice(0, 3)"
          :key="announcement.id"
          class="rounded-lg border border-gray-100 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="mb-2 flex items-center justify-between">
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ announcement.title }}</h3>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ formatDate(announcement.published_at || announcement.created_at) }}
            </span>
          </div>
          <!-- Render as iframe if URL -->
          <iframe
            v-if="announcement.content_type === 'url'"
            :src="announcement.content"
            class="h-48 w-full rounded-lg border-0"
            allowfullscreen
          ></iframe>
          <!-- Render as HTML/Markdown content -->
          <div
            v-else
            class="prose prose-sm max-w-none text-gray-600 dark:prose-invert dark:text-gray-300"
            v-html="announcement.content"
          ></div>
        </div>
      </div>

      <!-- Fallback to home_content if no announcements -->
      <div v-else-if="hasHomeContent">
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

      <!-- No Announcements -->
      <div v-else class="py-8 text-center text-sm text-gray-400">
        {{ t('dashboard.noAnnouncements') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { announcementAPI } from '@/api/announcement'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { Announcement } from '@/types'

const { t, locale } = useI18n()
const appStore = useAppStore()

const announcements = ref<Announcement[]>([])
const loading = ref(true)

// Fallback to home_content
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => !!homeContent.value.trim())
const isUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString(locale.value, { year: 'numeric', month: '2-digit', day: '2-digit' })
}

onMounted(async () => {
  try {
    announcements.value = await announcementAPI.getUnreadAnnouncements()
  } catch (e) {
    console.error('Failed to load announcements:', e)
  } finally {
    loading.value = false
  }
})
</script>
