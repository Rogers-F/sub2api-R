<template>
  <BaseDialog
    :show="show"
    :title="t('announcement.title')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
      </div>

      <!-- Empty state -->
      <div v-else-if="announcements.length === 0" class="text-center py-8">
        <Icon name="megaphone" size="xl" class="mx-auto text-gray-300 dark:text-dark-600" />
        <p class="mt-2 text-gray-500 dark:text-dark-400">{{ t('announcement.noAnnouncements') }}</p>
      </div>

      <!-- Announcements list -->
      <div v-else class="max-h-96 overflow-y-auto space-y-4">
        <div
          v-for="announcement in announcements"
          :key="announcement.id"
          class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800"
        >
          <h4 class="font-medium text-gray-900 dark:text-white">
            {{ announcement.title }}
          </h4>
          <div class="mt-2 text-sm text-gray-600 dark:text-dark-300">
            <!-- Markdown content -->
            <div
              v-if="announcement.content_type === 'markdown'"
              class="prose prose-sm dark:prose-invert max-w-none"
              v-html="renderMarkdown(announcement.content)"
            ></div>
            <!-- HTML content -->
            <div
              v-else-if="announcement.content_type === 'html'"
              class="prose prose-sm dark:prose-invert max-w-none"
              v-html="announcement.content"
            ></div>
            <!-- URL content -->
            <div v-else-if="announcement.content_type === 'url'" class="mt-2">
              <iframe
                :src="announcement.content"
                class="w-full h-64 rounded border border-gray-200 dark:border-dark-600"
                sandbox="allow-scripts allow-same-origin"
              ></iframe>
            </div>
          </div>
          <p v-if="announcement.published_at" class="mt-3 text-xs text-gray-400 dark:text-dark-500">
            {{ formatDate(announcement.published_at) }}
          </p>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button
          type="button"
          class="btn btn-primary"
          :disabled="marking"
          @click="handleMarkAllRead"
        >
          <span v-if="marking" class="flex items-center">
            <svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ t('common.loading') }}
          </span>
          <span v-else>{{ t('announcement.gotIt') }}</span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { announcementAPI } from '@/api/announcement'
import type { Announcement } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'read'): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()

const loading = ref(true)
const marking = ref(false)
const announcements = ref<Announcement[]>([])

const loadAnnouncements = async () => {
  loading.value = true
  try {
    announcements.value = await announcementAPI.getUnreadAnnouncements()
  } catch (error) {
    console.error('Failed to load announcements:', error)
  } finally {
    loading.value = false
  }
}

const handleMarkAllRead = async () => {
  if (announcements.value.length === 0) {
    emit('close')
    return
  }

  marking.value = true
  try {
    const ids = announcements.value.map(a => a.id)
    await announcementAPI.markAllAsRead(ids)
    emit('read')
    emit('close')
  } catch (error) {
    console.error('Failed to mark announcements as read:', error)
  } finally {
    marking.value = false
  }
}

const handleClose = () => {
  emit('close')
}

const renderMarkdown = (content: string): string => {
  // Simple markdown-like rendering: convert newlines to <br> and basic formatting
  return content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
    .replace(/`(.*?)`/g, '<code>$1</code>')
    .replace(/\n/g, '<br>')
}

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr)
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

onMounted(() => {
  loadAnnouncements()
})
</script>
