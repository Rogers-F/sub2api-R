<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex justify-end gap-3">
          <button
            @click="loadAnnouncements"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button @click="showCreateDialog = true" class="btn btn-primary">
            <Icon name="plus" size="md" class="mr-1" />
            {{ t('admin.announcements.create') }}
          </button>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="max-w-md flex-1">
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('common.search')"
              class="input"
              @input="handleSearch"
            />
          </div>
          <div class="flex gap-2">
            <Select
              v-model="filters.status"
              :options="filterStatusOptions"
              class="w-36"
              @change="loadAnnouncements"
            />
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="announcements" :loading="loading">
          <template #cell-title="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>

          <template #cell-content_type="{ value }">
            <span class="badge badge-gray">
              {{ t(`admin.announcements.contentType.${value}`) }}
            </span>
          </template>

          <template #cell-priority="{ value }">
            <span class="text-sm text-gray-600 dark:text-gray-300">{{ value }}</span>
          </template>

          <template #cell-status="{ value }">
            <span
              :class="[
                'badge',
                value === 'active' ? 'badge-success' : 'badge-gray'
              ]"
            >
              {{ t(`admin.announcements.status.${value}`) }}
            </span>
          </template>

          <template #cell-published_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ value ? formatDateTime(value) : '-' }}
            </span>
          </template>

          <template #cell-expires_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ value ? formatDateTime(value) : '-' }}
            </span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
              <button
                @click="handleEdit(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                @click="handleDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('common.delete')"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create Dialog -->
    <BaseDialog
      :show="showCreateDialog"
      :title="t('admin.announcements.create')"
      width="wide"
      @close="showCreateDialog = false"
    >
      <form id="create-announcement-form" @submit.prevent="handleCreate" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.announcements.form.title') }}</label>
          <input
            v-model="createForm.title"
            type="text"
            required
            class="input"
            :placeholder="t('admin.announcements.form.titlePlaceholder')"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.announcements.form.contentType') }}</label>
          <Select v-model="createForm.content_type" :options="contentTypeOptions" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.announcements.form.content') }}</label>
          <textarea
            v-model="createForm.content"
            rows="6"
            required
            class="input font-mono text-sm"
            :placeholder="t('admin.announcements.form.contentPlaceholder')"
          ></textarea>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.announcements.form.priority') }}</label>
            <input
              v-model.number="createForm.priority"
              type="number"
              min="0"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.announcements.form.priorityHint') }}
            </p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.status') }}</label>
            <Select v-model="createForm.status" :options="statusOptions" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">
              {{ t('admin.announcements.form.publishedAt') }}
              <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input
              v-model="createForm.published_at_str"
              type="datetime-local"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.announcements.form.publishedAtHint') }}
            </p>
          </div>
          <div>
            <label class="input-label">
              {{ t('admin.announcements.form.expiresAt') }}
              <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input
              v-model="createForm.expires_at_str"
              type="datetime-local"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.announcements.form.expiresAtHint') }}
            </p>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" @click="showCreateDialog = false" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="create-announcement-form" :disabled="creating" class="btn btn-primary">
            {{ creating ? t('common.creating') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog
      :show="showEditDialog"
      :title="t('admin.announcements.edit')"
      width="wide"
      @close="closeEditDialog"
    >
      <form id="edit-announcement-form" @submit.prevent="handleUpdate" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.announcements.form.title') }}</label>
          <input
            v-model="editForm.title"
            type="text"
            required
            class="input"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.announcements.form.contentType') }}</label>
          <Select v-model="editForm.content_type" :options="contentTypeOptions" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.announcements.form.content') }}</label>
          <textarea
            v-model="editForm.content"
            rows="6"
            required
            class="input font-mono text-sm"
          ></textarea>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.announcements.form.priority') }}</label>
            <input
              v-model.number="editForm.priority"
              type="number"
              min="0"
              class="input"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.status') }}</label>
            <Select v-model="editForm.status" :options="statusOptions" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">
              {{ t('admin.announcements.form.publishedAt') }}
            </label>
            <input
              v-model="editForm.published_at_str"
              type="datetime-local"
              class="input"
            />
          </div>
          <div>
            <label class="input-label">
              {{ t('admin.announcements.form.expiresAt') }}
            </label>
            <input
              v-model="editForm.expires_at_str"
              type="datetime-local"
              class="input"
            />
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" @click="closeEditDialog" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="edit-announcement-form" :disabled="updating" class="btn btn-primary">
            {{ updating ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.announcements.delete')"
      :message="t('admin.announcements.deleteConfirm')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAnnouncementAPI } from '@/api/announcement'
import { formatDateTime } from '@/utils/format'
import type { Announcement } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

// State
const announcements = ref<Announcement[]>([])
const loading = ref(false)
const creating = ref(false)
const updating = ref(false)
const searchQuery = ref('')

const filters = reactive({
  status: ''
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

// Dialogs
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)

const editingAnnouncement = ref<Announcement | null>(null)
const deletingAnnouncement = ref<Announcement | null>(null)

// Forms
const createForm = reactive({
  title: '',
  content: '',
  content_type: 'markdown' as 'markdown' | 'html' | 'url',
  priority: 0,
  status: 'active' as 'active' | 'inactive',
  published_at_str: '',
  expires_at_str: ''
})

const editForm = reactive({
  title: '',
  content: '',
  content_type: 'markdown' as 'markdown' | 'html' | 'url',
  priority: 0,
  status: 'active' as 'active' | 'inactive',
  published_at_str: '',
  expires_at_str: ''
})

// Options
const filterStatusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'active', label: t('admin.announcements.status.active') },
  { value: 'inactive', label: t('admin.announcements.status.inactive') }
])

const statusOptions = computed(() => [
  { value: 'active', label: t('admin.announcements.status.active') },
  { value: 'inactive', label: t('admin.announcements.status.inactive') }
])

const contentTypeOptions = computed(() => [
  { value: 'markdown', label: t('admin.announcements.contentType.markdown') },
  { value: 'html', label: t('admin.announcements.contentType.html') },
  { value: 'url', label: t('admin.announcements.contentType.url') }
])

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('admin.announcements.columns.title') },
  { key: 'content_type', label: t('admin.announcements.columns.contentType') },
  { key: 'priority', label: t('admin.announcements.columns.priority'), sortable: true },
  { key: 'status', label: t('admin.announcements.columns.status'), sortable: true },
  { key: 'published_at', label: t('admin.announcements.columns.publishedAt'), sortable: true },
  { key: 'expires_at', label: t('admin.announcements.columns.expiresAt'), sortable: true },
  { key: 'created_at', label: t('admin.announcements.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.announcements.columns.actions') }
])

// API calls
let abortController: AbortController | null = null

const loadAnnouncements = async () => {
  if (abortController) {
    abortController.abort()
  }
  const currentController = new AbortController()
  abortController = currentController
  loading.value = true

  try {
    const response = await adminAnnouncementAPI.list(
      pagination.page,
      pagination.page_size
    )
    if (currentController.signal.aborted) return

    announcements.value = response.items
    pagination.total = response.total
  } catch (error: any) {
    if (currentController.signal.aborted || error?.name === 'AbortError') return
    appStore.showError(t('admin.announcements.loadFailed'))
    console.error('Error loading announcements:', error)
  } finally {
    if (abortController === currentController && !currentController.signal.aborted) {
      loading.value = false
      abortController = null
    }
  }
}

let searchTimeout: ReturnType<typeof setTimeout>
const handleSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    loadAnnouncements()
  }, 300)
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadAnnouncements()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadAnnouncements()
}

// Helper to convert datetime-local to ISO 8601
const toISOString = (dateStr: string): string | undefined => {
  if (!dateStr) return undefined
  // datetime-local format: "YYYY-MM-DDTHH:mm"
  // Convert to ISO 8601: "YYYY-MM-DDTHH:mm:ss.000Z"
  return new Date(dateStr).toISOString()
}

// Create
const handleCreate = async () => {
  creating.value = true
  try {
    await adminAnnouncementAPI.create({
      title: createForm.title,
      content: createForm.content,
      content_type: createForm.content_type,
      priority: createForm.priority,
      status: createForm.status,
      published_at: toISOString(createForm.published_at_str),
      expires_at: toISOString(createForm.expires_at_str)
    })
    appStore.showSuccess(t('admin.announcements.createSuccess'))
    showCreateDialog.value = false
    resetCreateForm()
    loadAnnouncements()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.announcements.createFailed'))
  } finally {
    creating.value = false
  }
}

const resetCreateForm = () => {
  createForm.title = ''
  createForm.content = ''
  createForm.content_type = 'markdown'
  createForm.priority = 0
  createForm.status = 'active'
  createForm.published_at_str = ''
  createForm.expires_at_str = ''
}

// Edit
const handleEdit = (announcement: Announcement) => {
  editingAnnouncement.value = announcement
  editForm.title = announcement.title
  editForm.content = announcement.content
  editForm.content_type = announcement.content_type
  editForm.priority = announcement.priority
  editForm.status = announcement.status
  editForm.published_at_str = announcement.published_at ? new Date(announcement.published_at).toISOString().slice(0, 16) : ''
  editForm.expires_at_str = announcement.expires_at ? new Date(announcement.expires_at).toISOString().slice(0, 16) : ''
  showEditDialog.value = true
}

const closeEditDialog = () => {
  showEditDialog.value = false
  editingAnnouncement.value = null
}

const handleUpdate = async () => {
  if (!editingAnnouncement.value) return

  updating.value = true
  try {
    await adminAnnouncementAPI.update(editingAnnouncement.value.id, {
      title: editForm.title,
      content: editForm.content,
      content_type: editForm.content_type,
      priority: editForm.priority,
      status: editForm.status,
      published_at: toISOString(editForm.published_at_str),
      expires_at: toISOString(editForm.expires_at_str)
    })
    appStore.showSuccess(t('admin.announcements.updateSuccess'))
    closeEditDialog()
    loadAnnouncements()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.announcements.updateFailed'))
  } finally {
    updating.value = false
  }
}

// Delete
const handleDelete = (announcement: Announcement) => {
  deletingAnnouncement.value = announcement
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  if (!deletingAnnouncement.value) return

  try {
    await adminAnnouncementAPI.delete(deletingAnnouncement.value.id)
    appStore.showSuccess(t('admin.announcements.deleteSuccess'))
    showDeleteDialog.value = false
    deletingAnnouncement.value = null
    loadAnnouncements()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.announcements.deleteFailed'))
  }
}

onMounted(() => {
  loadAnnouncements()
})

onUnmounted(() => {
  clearTimeout(searchTimeout)
  abortController?.abort()
})
</script>
