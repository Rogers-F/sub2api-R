<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-3 lg:flex-row lg:items-center">
          <div class="flex flex-wrap items-center gap-3">
            <SearchInput
              v-model="searchQuery"
              :placeholder="t('admin.enterprises.search')"
              class="w-full sm:w-64"
              @search="loadEnterprises"
              @update:model-value="handleSearch"
            />
            <Select v-model="filters.status" class="w-40" :options="statusOptions" @change="loadEnterprises" />
          </div>
          <div class="flex flex-wrap justify-end gap-3">
            <button class="btn btn-secondary" :disabled="loading" @click="loadEnterprises">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button class="btn btn-primary" @click="openCreate">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.enterprises.create') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="enterprises" :loading="loading" row-key="id">
          <template #cell-name="{ row, value }">
            <button class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="openDetail(row.id)">
              {{ value }}
            </button>
          </template>
          <template #cell-status="{ value }">
            <span :class="['badge', value === 'active' ? 'badge-success' : 'badge-danger']">
              {{ enterpriseStatusLabel(value) }}
            </span>
          </template>
          <template #cell-notes="{ value }">
            <span v-if="value" class="block max-w-md truncate text-sm text-gray-600 dark:text-gray-300" :title="value">{{ value }}</span>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>
          <template #cell-account_count="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value || 0 }}</span>
          </template>
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ formatRelativeTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700" @click="openEdit(row)">
                <Icon name="edit" size="sm" />
              </button>
              <button
                class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700"
                @click="toggleStatus(row)"
              >
                <Icon :name="row.status === 'active' ? 'ban' : 'play'" size="sm" />
              </button>
              <button class="rounded-lg p-1.5 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20" @click="openDelete(row)">
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

    <BaseDialog :show="showModal" :title="editingEnterprise ? t('admin.enterprises.edit') : t('admin.enterprises.create')" width="normal" @close="closeModal">
      <form id="enterprise-form" class="space-y-4" @submit.prevent="saveEnterprise">
        <div>
          <label class="input-label">{{ t('common.name') }}</label>
          <input v-model="form.name" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('admin.enterprises.notes') }}</label>
          <textarea v-model="form.notes" class="input" rows="3" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.enterprises.status') }}</label>
          <Select v-model="form.status" :options="statusEditOptions" />
        </div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="closeModal">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" type="submit" form="enterprise-form" :disabled="saving">
          {{ t('common.save') }}
        </button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDelete"
      :title="t('admin.enterprises.delete')"
      :message="t('admin.enterprises.deleteConfirm', { name: deletingEnterprise?.name || '' })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDelete = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatRelativeTime } from '@/utils/format'
import type { Enterprise } from '@/types'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const enterprises = ref<Enterprise[]>([])
const loading = ref(false)
const saving = ref(false)
const searchQuery = ref('')
const filters = reactive({ status: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0, pages: 0 })

const showModal = ref(false)
const showDelete = ref(false)
const editingEnterprise = ref<Enterprise | null>(null)
const deletingEnterprise = ref<Enterprise | null>(null)
const form = reactive({ name: '', notes: '', status: 'active' as 'active' | 'disabled' })

const columns = computed(() => [
  { key: 'name', label: t('admin.enterprises.columns.name'), sortable: true },
  { key: 'status', label: t('admin.enterprises.columns.status'), sortable: true },
  { key: 'notes', label: t('admin.enterprises.columns.notes'), sortable: false },
  { key: 'account_count', label: t('admin.enterprises.columns.accountCount'), sortable: true },
  { key: 'created_at', label: t('admin.enterprises.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.enterprises.columns.actions'), sortable: false }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.enterprises.allStatus') },
  { value: 'active', label: t('admin.enterprises.statusActive') },
  { value: 'disabled', label: t('admin.enterprises.statusDisabled') }
])

const statusEditOptions = computed(() => [
  { value: 'active', label: t('admin.enterprises.statusActive') },
  { value: 'disabled', label: t('admin.enterprises.statusDisabled') }
])

const enterpriseStatusLabel = (status: string) => status === 'active' ? t('admin.enterprises.statusActive') : t('admin.enterprises.statusDisabled')

let searchTimer: number | undefined
const handleSearch = () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    pagination.page = 1
    loadEnterprises()
  }, 300)
}

const loadEnterprises = async () => {
  loading.value = true
  try {
    const result = await adminAPI.enterprises.list(pagination.page, pagination.page_size, {
      search: searchQuery.value,
      status: filters.status
    })
    enterprises.value = result.items || []
    pagination.total = result.total || 0
    pagination.pages = result.pages || 0
  } catch (error) {
    console.error('Failed to load enterprises:', error)
    appStore.showError(t('admin.enterprises.loadFailed'))
  } finally {
    loading.value = false
  }
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadEnterprises()
}

const handlePageSizeChange = (size: number) => {
  pagination.page = 1
  pagination.page_size = size
  loadEnterprises()
}

const openCreate = () => {
  editingEnterprise.value = null
  form.name = ''
  form.notes = ''
  form.status = 'active'
  showModal.value = true
}

const openEdit = (enterprise: Enterprise) => {
  editingEnterprise.value = enterprise
  form.name = enterprise.name
  form.notes = enterprise.notes || ''
  form.status = enterprise.status
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingEnterprise.value = null
}

const saveEnterprise = async () => {
  saving.value = true
  try {
    const payload = { name: form.name.trim(), notes: form.notes.trim() || null, status: form.status }
    if (editingEnterprise.value) {
      await adminAPI.enterprises.update(editingEnterprise.value.id, payload)
      appStore.showSuccess(t('admin.enterprises.updated'))
    } else {
      await adminAPI.enterprises.create(payload)
      appStore.showSuccess(t('admin.enterprises.created'))
    }
    closeModal()
    await loadEnterprises()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.enterprises.saveFailed'))
  } finally {
    saving.value = false
  }
}

const toggleStatus = async (enterprise: Enterprise) => {
  try {
    await adminAPI.enterprises.update(enterprise.id, {
      name: enterprise.name,
      notes: enterprise.notes || null,
      status: enterprise.status === 'active' ? 'disabled' : 'active'
    })
    await loadEnterprises()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.enterprises.saveFailed'))
  }
}

const openDelete = (enterprise: Enterprise) => {
  deletingEnterprise.value = enterprise
  showDelete.value = true
}

const confirmDelete = async () => {
  if (!deletingEnterprise.value) return
  try {
    await adminAPI.enterprises.delete(deletingEnterprise.value.id)
    appStore.showSuccess(t('admin.enterprises.deleted'))
    showDelete.value = false
    deletingEnterprise.value = null
    await loadEnterprises()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.enterprises.deleteFailed'))
  }
}

const openDetail = (id: number) => {
  router.push(`/admin/enterprises/${id}`)
}

onMounted(loadEnterprises)
</script>
