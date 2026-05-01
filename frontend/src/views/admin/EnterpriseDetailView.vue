<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div class="flex flex-col justify-between gap-3 lg:flex-row lg:items-start">
            <div>
              <button class="mb-2 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="router.push('/admin/enterprises')">
                {{ t('admin.enterprises.back') }}
              </button>
              <div class="flex flex-wrap items-center gap-3">
                <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ enterprise?.name || '-' }}</h1>
                <span v-if="enterprise" :class="['badge', enterprise.status === 'active' ? 'badge-success' : 'badge-danger']">
                  {{ enterprise.status === 'active' ? t('admin.enterprises.statusActive') : t('admin.enterprises.statusDisabled') }}
                </span>
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.enterprises.accountCount', { count: enterprise?.account_count || 0 }) }}
                </span>
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  RPM {{ formatMetricNumber(enterprise?.rpm) }}
                </span>
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.enterprises.errorRate5m') }} {{ formatErrorRate(enterprise?.error_rate_5m) }}
                </span>
              </div>
              <p v-if="enterprise?.notes" class="mt-2 max-w-3xl text-sm text-gray-600 dark:text-gray-300">{{ enterprise.notes }}</p>
            </div>
            <div class="flex flex-wrap justify-end gap-3">
              <button class="btn btn-secondary" :disabled="loading" @click="reloadAll">
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
              <button class="btn btn-secondary" :disabled="enterprise?.status !== 'active'" @click="openMoveIn">
                <Icon name="plus" size="md" class="mr-2" />
                {{ t('admin.enterprises.moveIn') }}
              </button>
              <button class="btn btn-secondary" :disabled="selectedIds.length === 0" @click="moveSelectedOut">
                {{ t('admin.enterprises.moveOut') }}
              </button>
              <button class="btn btn-primary" :disabled="enterprise?.status !== 'active'" @click="showCreate = true">
                <Icon name="plus" size="md" class="mr-2" />
                {{ t('admin.accounts.createAccount') }}
              </button>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <SearchInput v-model="filters.search" class="w-full sm:w-64" :placeholder="t('admin.accounts.searchAccounts')" @search="loadAccounts" @update:model-value="handleSearch" />
            <Select v-model="filters.platform" class="w-40" :options="platformOptions" @change="loadAccounts" />
            <Select v-model="filters.status" class="w-40" :options="statusOptions" @change="loadAccounts" />
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="accounts" :loading="loading" row-key="id">
          <template #header-select>
            <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" :checked="allSelected" @change="toggleAll" />
          </template>
          <template #cell-select="{ row }">
            <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" :checked="selectedIds.includes(row.id)" @change="toggleSelected(row.id)" />
          </template>
          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>
          <template #cell-platform_type="{ row }">
            <PlatformTypeBadge :platform="row.platform" :type="row.type" :plan-type="row.credentials?.plan_type" :privacy-mode="row.extra?.privacy_mode" />
          </template>
          <template #cell-status="{ row }">
            <AccountStatusIndicator :account="row" />
          </template>
          <template #cell-schedulable="{ row }">
            <button
              class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800"
              :class="row.schedulable ? 'bg-primary-500 hover:bg-primary-600' : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500'"
              :disabled="togglingSchedulable === row.id"
              :title="row.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')"
              @click="toggleSchedulable(row)"
            >
              <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out" :class="row.schedulable ? 'translate-x-4' : 'translate-x-0'" />
            </button>
          </template>
          <template #cell-groups="{ row }">
            <AccountGroupsCell :groups="row.groups" :max-display="3" />
          </template>
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ formatRelativeTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <button class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700" @click="openEdit(row)">
              <Icon name="edit" size="sm" />
            </button>
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

    <CreateAccountModal
      :show="showCreate"
      :proxies="proxies"
      :groups="groups"
      :enterprises="activeEnterprises"
      :default-enterprise-id="enterprise?.status === 'active' ? enterprise.id : null"
      @close="showCreate = false"
      @created="handleAccountChanged"
    />
    <EditAccountModal
      :show="showEdit"
      :account="editingAccount"
      :proxies="proxies"
      :groups="groups"
      :enterprises="activeEnterprises"
      @close="showEdit = false"
      @updated="handleAccountChanged"
    />

    <BaseDialog :show="showMoveIn" :title="t('admin.enterprises.moveIn')" width="wide" @close="showMoveIn = false">
      <div class="space-y-4">
        <div class="flex flex-wrap items-center gap-3">
          <SearchInput v-model="moveFilters.search" class="w-full sm:w-64" :placeholder="t('admin.accounts.searchAccounts')" @search="loadMoveCandidates" @update:model-value="handleMoveSearch" />
          <Select v-model="moveFilters.enterprise" class="w-44" :options="moveEnterpriseOptions" @change="loadMoveCandidates" />
        </div>
        <DataTable :columns="moveColumns" :data="moveCandidates" :loading="moveLoading" row-key="id">
          <template #cell-select="{ row }">
            <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" :checked="moveIds.includes(row.id)" @change="toggleMoveSelected(row.id)" />
          </template>
          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>
          <template #cell-enterprise="{ row }">
            <span class="text-sm text-gray-600 dark:text-gray-300">{{ row.enterprise?.name || t('admin.enterprises.unassigned') }}</span>
          </template>
        </DataTable>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="showMoveIn = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="moveIds.length === 0 || moving" @click="moveSelectedIn">
          {{ t('admin.enterprises.moveSelectedIn', { count: moveIds.length }) }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import { CreateAccountModal, EditAccountModal } from '@/components/account'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import { formatRelativeTime } from '@/utils/format'
import type { Account, AdminGroup, Enterprise, Proxy as AccountProxy } from '@/types'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const enterpriseId = computed(() => Number(route.params.id))
const enterprise = ref<Enterprise | null>(null)
const accounts = ref<Account[]>([])
const proxies = ref<AccountProxy[]>([])
const groups = ref<AdminGroup[]>([])
const activeEnterprises = ref<Enterprise[]>([])
const loading = ref(false)
const moving = ref(false)
const selectedIds = ref<number[]>([])
const showCreate = ref(false)
const showEdit = ref(false)
const editingAccount = ref<Account | null>(null)
const showMoveIn = ref(false)
const moveCandidates = ref<Account[]>([])
const moveIds = ref<number[]>([])
const moveLoading = ref(false)
const togglingSchedulable = ref<number | null>(null)

const filters = reactive({ search: '', platform: '', status: '' })
const moveFilters = reactive({ search: '', enterprise: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0, pages: 0 })

const platformOptions = computed(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.accounts.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'error', label: t('admin.accounts.status.error') }
])

const moveEnterpriseOptions = computed(() => [
  { value: '', label: t('admin.enterprises.allAccounts') },
  { value: 'unassigned', label: t('admin.enterprises.unassigned') }
])

const columns = computed(() => [
  { key: 'select', label: '', sortable: false },
  { key: 'name', label: t('admin.accounts.columns.name'), sortable: true },
  { key: 'platform_type', label: t('admin.accounts.columns.platformType'), sortable: false },
  { key: 'status', label: t('admin.accounts.columns.status'), sortable: true },
  { key: 'schedulable', label: t('admin.accounts.columns.schedulable'), sortable: true },
  { key: 'groups', label: t('admin.accounts.columns.groups'), sortable: false },
  { key: 'created_at', label: t('admin.accounts.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.accounts.columns.actions'), sortable: false }
])

const moveColumns = computed(() => [
  { key: 'select', label: '', sortable: false },
  { key: 'name', label: t('admin.accounts.columns.name'), sortable: true },
  { key: 'enterprise', label: t('admin.enterprises.title'), sortable: false }
])

const allSelected = computed(() => accounts.value.length > 0 && accounts.value.every(account => selectedIds.value.includes(account.id)))

const formatMetricNumber = (value: unknown) => {
  const n = typeof value === 'number' && Number.isFinite(value) ? value : 0
  return Math.round(n).toLocaleString()
}

const formatErrorRate = (value: unknown) => {
  const n = typeof value === 'number' && Number.isFinite(value) ? value : 0
  return `${(n * 100).toFixed(2)}%`
}

let searchTimer: number | undefined
let moveSearchTimer: number | undefined

const loadEnterprise = async () => {
  enterprise.value = await adminAPI.enterprises.getById(enterpriseId.value)
}

const loadAccounts = async () => {
  loading.value = true
  try {
    const result = await adminAPI.enterprises.listAccounts(enterpriseId.value, pagination.page, pagination.page_size, filters)
    accounts.value = result.items || []
    pagination.total = result.total || 0
    pagination.pages = result.pages || 0
    selectedIds.value = selectedIds.value.filter(id => accounts.value.some(account => account.id === id))
  } catch (error) {
    console.error('Failed to load enterprise accounts:', error)
    appStore.showError(t('admin.enterprises.accountsLoadFailed'))
  } finally {
    loading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([loadEnterprise(), loadAccounts(), loadLookups()])
}

const loadLookups = async () => {
  const [proxyList, groupList, enterpriseList] = await Promise.all([
    adminAPI.proxies.getAll(),
    adminAPI.groups.getAll(),
    adminAPI.enterprises.listActive()
  ])
  proxies.value = proxyList
  groups.value = groupList
  activeEnterprises.value = enterpriseList
}

const loadMoveCandidates = async () => {
  moveLoading.value = true
  try {
    const result = await adminAPI.accounts.list(1, 50, {
      search: moveFilters.search,
      enterprise: moveFilters.enterprise
    })
    moveCandidates.value = result.items || []
    moveIds.value = moveIds.value.filter(id => moveCandidates.value.some(account => account.id === id))
  } catch (error) {
    console.error('Failed to load move candidates:', error)
    appStore.showError(t('admin.enterprises.accountsLoadFailed'))
  } finally {
    moveLoading.value = false
  }
}

const handleSearch = () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    pagination.page = 1
    loadAccounts()
  }, 300)
}

const handleMoveSearch = () => {
  window.clearTimeout(moveSearchTimer)
  moveSearchTimer = window.setTimeout(loadMoveCandidates, 300)
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadAccounts()
}

const handlePageSizeChange = (size: number) => {
  pagination.page = 1
  pagination.page_size = size
  loadAccounts()
}

const toggleSelected = (id: number) => {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter(item => item !== id)
    : [...selectedIds.value, id]
}

const toggleAll = () => {
  selectedIds.value = allSelected.value ? [] : accounts.value.map(account => account.id)
}

const toggleMoveSelected = (id: number) => {
  moveIds.value = moveIds.value.includes(id)
    ? moveIds.value.filter(item => item !== id)
    : [...moveIds.value, id]
}

const moveSelectedIn = async () => {
  moving.value = true
  try {
    await adminAPI.enterprises.assignAccounts(enterpriseId.value, moveIds.value)
    appStore.showSuccess(t('admin.enterprises.movedIn'))
    showMoveIn.value = false
    moveIds.value = []
    await reloadAll()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.enterprises.moveFailed'))
  } finally {
    moving.value = false
  }
}

const openMoveIn = async () => {
  showMoveIn.value = true
  await loadMoveCandidates()
}

const moveSelectedOut = async () => {
  moving.value = true
  try {
    await adminAPI.enterprises.unassignAccounts(enterpriseId.value, selectedIds.value)
    appStore.showSuccess(t('admin.enterprises.movedOut'))
    selectedIds.value = []
    await reloadAll()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.enterprises.moveFailed'))
  } finally {
    moving.value = false
  }
}

const openEdit = (account: Account) => {
  editingAccount.value = account
  showEdit.value = true
}

const handleAccountChanged = async () => {
  showCreate.value = false
  showEdit.value = false
  editingAccount.value = null
  await reloadAll()
}

const toggleSchedulable = async (account: Account) => {
  const nextSchedulable = !account.schedulable
  togglingSchedulable.value = account.id
  try {
    const updated = await adminAPI.accounts.setSchedulable(account.id, nextSchedulable)
    accounts.value = accounts.value.map((item) => (
      item.id === account.id ? { ...item, schedulable: updated?.schedulable ?? nextSchedulable } : item
    ))
  } catch (error) {
    console.error('Failed to toggle enterprise account schedulable:', error)
    appStore.showError(t('admin.accounts.failedToToggleSchedulable'))
  } finally {
    togglingSchedulable.value = null
  }
}

onMounted(async () => {
  await reloadAll()
  await loadMoveCandidates()
})
</script>
