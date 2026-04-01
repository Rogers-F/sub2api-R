<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('admin.payg.title') }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.payg.description') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadWallet">
          <Icon name="refresh" size="sm" class="mr-1.5" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-16">
        <div class="h-10 w-10 animate-spin rounded-full border-b-2 border-primary-500"></div>
      </div>

      <template v-else-if="wallet">
        <div
          v-if="!wallet.enabled"
          class="rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationTriangle" size="md" class="mt-0.5 text-amber-500" />
            <div>
              <div class="font-medium">{{ t('admin.payg.disabledTitle') }}</div>
              <div class="mt-1">{{ t('admin.payg.disabledDescription') }}</div>
            </div>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.payg.totalOrders') }}</div>
            <div class="mt-2 text-3xl font-bold text-gray-900 dark:text-white">{{ wallet.total_orders }}</div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.payg.paidOrders') }}</div>
            <div class="mt-2 text-3xl font-bold text-emerald-600 dark:text-emerald-400">{{ wallet.paid_orders }}</div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.payg.pendingOrders') }}</div>
            <div class="mt-2 text-3xl font-bold text-amber-600 dark:text-amber-400">{{ wallet.pending_orders }}</div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.payg.totalPaidAmount') }}</div>
            <div class="mt-2 text-3xl font-bold text-gray-900 dark:text-white">{{ formatCny(wallet.total_paid_amount) }}</div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.payg.totalCreditedAmount') }}</div>
            <div class="mt-2 text-3xl font-bold text-gray-900 dark:text-white">{{ formatUsd(wallet.total_credited_amount) }}</div>
          </div>
        </div>

        <div class="grid gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
          <div class="card flex h-full flex-col overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.payg.userRanking') }}
                  </h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.payg.rankingSummary', { count: rankingTotal }) }}
                  </p>
                </div>
                <span
                  class="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                >
                  {{ rankingTotal }}
                </span>
              </div>
            </div>
            <div class="flex flex-1 flex-col p-6">
              <div v-if="rankingTotal === 0" class="flex flex-1 items-center justify-center py-12 text-center">
                <Icon name="users" size="xl" class="mx-auto text-gray-300 dark:text-dark-600" />
                <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.payg.noUsers') }}
                </p>
              </div>
              <div
                v-else
                class="flex flex-1 flex-col overflow-hidden rounded-xl border border-gray-100 dark:border-dark-700"
              >
                <div class="overflow-x-auto">
                  <table class="w-full min-w-[520px]">
                    <thead>
                      <tr class="border-b border-gray-200 text-left text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
                        <th class="px-3 py-3">{{ t('common.email') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.orderCount') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.totalPaidAmount') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.totalCreditedAmount') }}</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                      <tr
                        v-for="item in pagedUsers"
                        :key="item.user_id"
                        class="hover:bg-gray-50 dark:hover:bg-dark-800/60"
                      >
                        <td class="px-3 py-3 text-sm text-gray-900 dark:text-white">
                          <div>{{ item.email }}</div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">UID: {{ item.user_id }}</div>
                        </td>
                        <td class="px-3 py-3 text-sm text-gray-700 dark:text-gray-300">{{ item.order_count }}</td>
                        <td class="px-3 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatCny(item.total_paid_amount) }}</td>
                        <td class="px-3 py-3 text-sm font-medium text-emerald-600 dark:text-emerald-400">{{ formatUsd(item.total_credited_amount) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <Pagination
                  :page="rankingPage"
                  :total="rankingTotal"
                  :page-size="rankingPageSize"
                  :page-size-options="pageSizeOptions"
                  show-jump
                  @update:page="handleRankingPageChange"
                  @update:pageSize="handleRankingPageSizeChange"
                />
              </div>
            </div>
          </div>

          <div class="card flex h-full flex-col overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.payg.recentOrders') }}
                  </h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.payg.ordersSummary', { count: ordersTotal }) }}
                  </p>
                </div>
                <span
                  class="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                >
                  {{ ordersTotal }}
                </span>
              </div>
            </div>
            <div class="flex flex-1 flex-col p-6">
              <div v-if="ordersTotal === 0" class="flex flex-1 items-center justify-center py-12 text-center">
                <Icon name="list" size="xl" class="mx-auto text-gray-300 dark:text-dark-600" />
                <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.payg.noOrders') }}
                </p>
              </div>
              <div
                v-else
                class="flex flex-1 flex-col overflow-hidden rounded-xl border border-gray-100 dark:border-dark-700"
              >
                <div class="overflow-x-auto">
                  <table class="w-full min-w-[760px]">
                    <thead>
                      <tr class="border-b border-gray-200 text-left text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
                        <th class="px-3 py-3">{{ t('common.email') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.orderNo') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.paymentMethod') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.orderAmount') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.creditAmount') }}</th>
                        <th class="px-3 py-3">{{ t('common.status') }}</th>
                        <th class="px-3 py-3">{{ t('admin.payg.createdAt') }}</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                      <tr
                        v-for="order in pagedOrders"
                        :key="order.id"
                        class="hover:bg-gray-50 dark:hover:bg-dark-800/60"
                      >
                        <td class="px-3 py-3 text-sm text-gray-900 dark:text-white">{{ order.email }}</td>
                        <td class="px-3 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{{ order.client_sn }}</td>
                        <td class="px-3 py-3 text-sm text-gray-700 dark:text-gray-300">{{ order.payway_name || orderPaywayLabel(order.payway) }}</td>
                        <td class="px-3 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatCny(order.amount_yuan) }}</td>
                        <td class="px-3 py-3 text-sm font-medium text-emerald-600 dark:text-emerald-400">{{ formatUsd(order.credit_amount) }}</td>
                        <td class="px-3 py-3">
                          <span :class="['badge', orderStatusClass(order.status)]">
                            {{ orderStatusLabel(order.status) }}
                          </span>
                        </td>
                        <td class="px-3 py-3 text-sm text-gray-500 dark:text-gray-400">{{ formatDateTime(order.created_at) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <Pagination
                  :page="ordersPage"
                  :total="ordersTotal"
                  :page-size="ordersPageSize"
                  :page-size-options="pageSizeOptions"
                  show-jump
                  @update:page="handleOrdersPageChange"
                  @update:pageSize="handleOrdersPageSizeChange"
                />
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PaygAdminWallet } from '@/types'
import { adminAPI } from '@/api/admin'
import Pagination from '@/components/common/Pagination.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores'
import { formatCurrency, formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const wallet = ref<PaygAdminWallet | null>(null)
const pageSizeOptions = [10, 20, 50]

const rankingPage = ref(1)
const rankingPageSize = ref(normalizePageSize(getPersistedPageSize(10)))
const ordersPage = ref(1)
const ordersPageSize = ref(normalizePageSize(getPersistedPageSize(10)))

const rankingTotal = computed(() => wallet.value?.users.length ?? 0)
const ordersTotal = computed(() => wallet.value?.orders.length ?? 0)

const pagedUsers = computed(() =>
  sliceItems(wallet.value?.users ?? [], rankingPage.value, rankingPageSize.value)
)

const pagedOrders = computed(() =>
  sliceItems(wallet.value?.orders ?? [], ordersPage.value, ordersPageSize.value)
)

function normalizePageSize(value: number): number {
  return pageSizeOptions.includes(value) ? value : pageSizeOptions[0]
}

function sliceItems<T>(items: T[], page: number, pageSize: number): T[] {
  const start = (page - 1) * pageSize
  return items.slice(start, start + pageSize)
}

function getLastPage(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize))
}

function clampPage(pageRef: { value: number }, total: number, pageSize: number): void {
  pageRef.value = Math.min(Math.max(pageRef.value, 1), getLastPage(total, pageSize))
}

function handleRankingPageChange(page: number): void {
  rankingPage.value = page
}

function handleRankingPageSizeChange(pageSize: number): void {
  rankingPageSize.value = normalizePageSize(pageSize)
  rankingPage.value = 1
}

function handleOrdersPageChange(page: number): void {
  ordersPage.value = page
}

function handleOrdersPageSizeChange(pageSize: number): void {
  ordersPageSize.value = normalizePageSize(pageSize)
  ordersPage.value = 1
}

function formatUsd(value: number): string {
  return formatCurrency(value)
}

function formatCny(value: number): string {
  return formatCurrency(value, 'CNY')
}

function orderStatusLabel(status: string): string {
  switch (status) {
    case 'PAID':
      return t('admin.payg.statusPaid')
    case 'CLOSED':
      return t('admin.payg.statusClosed')
    default:
      return t('admin.payg.statusPending')
  }
}

function orderStatusClass(status: string): string {
  switch (status) {
    case 'PAID':
      return 'badge-success'
    case 'CLOSED':
      return 'badge-danger'
    default:
      return 'badge-warning'
  }
}

function orderPaywayLabel(code: string): string {
  return code === '3' ? t('admin.payg.wechat') : t('admin.payg.alipay')
}

async function loadWallet(): Promise<void> {
  loading.value = true
  try {
    wallet.value = await adminAPI.payg.getWallet()
    clampPage(rankingPage, rankingTotal.value, rankingPageSize.value)
    clampPage(ordersPage, ordersTotal.value, ordersPageSize.value)
  } catch (error: any) {
    appStore.showError(
      t('admin.payg.loadFailed') + ': ' + (error.message || t('common.unknownError'))
    )
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadWallet()
})

watch(rankingTotal, (total) => {
  clampPage(rankingPage, total, rankingPageSize.value)
})

watch(ordersTotal, (total) => {
  clampPage(ordersPage, total, ordersPageSize.value)
})
</script>
