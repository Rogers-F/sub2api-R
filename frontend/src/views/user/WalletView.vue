<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-primary-fg">
            {{ t('wallet.title') }}
          </h1>
          <p class="text-sm text-secondary-fg">
            {{ t('wallet.description') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadWallet()">
          <Icon name="refresh" size="sm" class="mr-1.5" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-16">
        <div class="h-10 w-10 animate-spin rounded-full border-b-2 border-gold-500 dark:border-gold-300"></div>
      </div>

      <template v-else-if="wallet">
        <div
          v-if="!wallet.enabled"
          class="rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationTriangle" size="md" class="mt-0.5 text-amber-500" />
            <div>
              <div class="font-medium">{{ t('wallet.disabledTitle') }}</div>
              <div class="mt-1">{{ t('wallet.disabledDescription') }}</div>
            </div>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div class="card p-5">
            <div class="text-sm text-secondary-fg">{{ t('wallet.balance') }}</div>
            <div class="mt-2 text-3xl font-bold num-mono text-primary-fg">
              {{ formatUsd(wallet.balance) }}
            </div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-secondary-fg">{{ t('wallet.totalPaid') }}</div>
            <div class="mt-2 text-3xl font-bold num-mono text-primary-fg">
              {{ formatCny(wallet.total_paid_amount) }}
            </div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-secondary-fg">{{ t('wallet.totalCredited') }}</div>
            <div class="mt-2 text-3xl font-bold num-mono text-primary-fg">
              {{ formatUsd(wallet.total_credited_amount) }}
            </div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-secondary-fg">{{ t('wallet.totalConsumption') }}</div>
            <div class="mt-2 text-3xl font-bold num-mono text-primary-fg">
              {{ formatUsd(wallet.total_consumption) }}
            </div>
          </div>
        </div>

        <div class="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(360px,0.9fr)]">
          <div class="card">
            <div class="hairline border-b px-6 py-4">
              <h2 class="text-lg font-semibold text-primary-fg">
                {{ t('wallet.rechargeTitle') }}
              </h2>
              <p class="mt-1 text-sm text-secondary-fg">
                {{ t('wallet.rechargeDescription', { rate: wallet.exchange_rate.toFixed(2) }) }}
              </p>
            </div>
            <div class="space-y-6 p-6">
              <div class="rounded-xl border border-blue-100 bg-blue-50 px-4 py-3 text-sm text-blue-700 dark:border-blue-900/60 dark:bg-blue-900/20 dark:text-blue-300">
                {{ t('wallet.builtInHint') }}
              </div>

              <div>
                <div class="mb-3 text-sm font-medium text-dust-700 dark:text-pearl-100">
                  {{ t('wallet.fixedAmounts') }}
                </div>
                <div class="flex flex-wrap gap-2">
                  <button
                    v-for="amount in fixedAmountOptions"
                    :key="amount"
                    type="button"
                    :class="[
                      'rounded-xl border px-4 py-2 text-sm font-medium transition',
                      selectedAmount === amount
                        ? 'border-gold-500 dark:border-gold-300 bg-gold-500/10 dark:bg-gold-300/10 text-gold-600 dark:text-gold-300'
                        : 'border-paper-200 dark:border-ink-700 text-dust-700 dark:text-pearl-100 hover:border-gold-500/50 dark:hover:border-gold-300/50 hover:text-gold-600 dark:hover:text-gold-300'
                    ]"
                    @click="selectFixedAmount(amount)"
                  >
                    {{ formatCny(amount) }}
                  </button>
                </div>
              </div>

              <div>
                <label class="mb-2 block text-sm font-medium text-dust-700 dark:text-pearl-100">
                  {{ t('wallet.customAmount') }}
                </label>
                <input
                  v-model="customAmount"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input w-full"
                  :placeholder="t('wallet.customAmountPlaceholder')"
                  @input="selectedAmount = null"
                />
              </div>

              <div>
                <div class="mb-3 text-sm font-medium text-dust-700 dark:text-pearl-100">
                  {{ t('wallet.paymentMethod') }}
                </div>
                <div class="grid gap-3 sm:grid-cols-2">
                  <button
                    type="button"
                    :class="payway === PAYWAY_ALIPAY ? activePaywayClass : inactivePaywayClass"
                    @click="payway = PAYWAY_ALIPAY"
                  >
                    {{ t('wallet.alipay') }}
                  </button>
                  <button
                    type="button"
                    :class="payway === PAYWAY_WECHAT ? activePaywayClass : inactivePaywayClass"
                    @click="payway = PAYWAY_WECHAT"
                  >
                    {{ t('wallet.wechat') }}
                  </button>
                </div>
              </div>

              <div class="rounded-xl hairline-strong border bg-paper-50 px-4 py-3 text-sm dark:bg-ink-800">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-secondary-fg">{{ t('wallet.selectedAmount') }}</span>
                  <span class="font-semibold num-mono text-primary-fg">
                    {{ effectiveAmount > 0 ? formatCny(effectiveAmount) : t('wallet.noAmountSelected') }}
                  </span>
                </div>
                <div class="mt-2 flex items-center justify-between gap-3">
                  <span class="text-secondary-fg">{{ t('wallet.creditPreview') }}</span>
                  <span class="font-semibold num-mono text-mint-600 dark:text-mint-500">
                    {{ effectiveAmount > 0 ? formatUsd(effectiveAmount * wallet.exchange_rate) : formatUsd(0) }}
                  </span>
                </div>
              </div>

              <div class="flex justify-end">
                <button
                  type="button"
                  class="btn btn-primary"
                  :disabled="creatingOrder || !wallet.enabled"
                  @click="createOrder"
                >
                  <svg
                    v-if="creatingOrder"
                    class="mr-2 h-4 w-4 animate-spin"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                    ></path>
                  </svg>
                  {{ creatingOrder ? t('wallet.creatingOrder') : t('wallet.createOrder') }}
                </button>
              </div>
            </div>
          </div>

          <div class="card">
            <div class="hairline border-b px-6 py-4">
              <h2 class="text-lg font-semibold text-primary-fg">
                {{ t('wallet.currentOrder') }}
              </h2>
              <p class="mt-1 text-sm text-secondary-fg">
                {{ t('wallet.currentOrderHint') }}
              </p>
            </div>
            <div class="p-6">
              <div v-if="activeOrder" class="space-y-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <div class="text-sm text-secondary-fg">{{ t('wallet.orderNo') }}</div>
                    <div class="font-mono text-sm text-primary-fg">{{ activeOrder.client_sn }}</div>
                  </div>
                  <span :class="['badge', orderStatusClass(activeOrder.status)]">
                    {{ orderStatusLabel(activeOrder.status) }}
                  </span>
                </div>

                <div class="grid gap-3 sm:grid-cols-2">
                  <div class="rounded-xl hairline-strong border p-4">
                    <div class="text-xs text-secondary-fg">{{ t('wallet.orderAmount') }}</div>
                    <div class="mt-1 text-lg font-semibold num-mono text-primary-fg">
                      {{ formatCny(activeOrder.amount_yuan) }}
                    </div>
                  </div>
                  <div class="rounded-xl hairline-strong border p-4">
                    <div class="text-xs text-secondary-fg">{{ t('wallet.creditAmount') }}</div>
                    <div class="mt-1 text-lg font-semibold num-mono text-mint-600 dark:text-mint-500">
                      {{ formatUsd(activeOrder.credit_amount) }}
                    </div>
                  </div>
                </div>

                <div class="rounded-2xl hairline-strong border bg-paper-50 p-4 dark:bg-ink-800">
                  <div v-if="activeQRCodeDataUrl" class="flex flex-col items-center">
                    <img :src="activeQRCodeDataUrl" :alt="t('wallet.qrCodeAlt')" class="h-64 w-64 rounded-xl bg-white p-3" />
                    <p class="mt-3 text-center text-xs text-secondary-fg">
                      {{ t('wallet.qrCodeHint') }}
                    </p>
                  </div>
                  <div v-else class="py-8 text-center text-sm text-secondary-fg">
                    {{ t('wallet.qrCodeUnavailable') }}
                  </div>
                </div>

                <div class="flex flex-wrap justify-end gap-2">
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm"
                    :disabled="!activePaymentCode"
                    @click="copyPaymentCode"
                  >
                    <Icon name="copy" size="sm" class="mr-1.5" />
                    {{ t('wallet.copyPaymentCode') }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm"
                    :disabled="syncingOrder"
                    @click="refreshActiveOrder"
                  >
                    <Icon name="refresh" size="sm" class="mr-1.5" />
                    {{ syncingOrder ? t('wallet.syncingOrder') : t('wallet.refreshOrder') }}
                  </button>
                </div>
              </div>

              <div v-else class="py-12 text-center">
                <Icon name="qrCode" size="xl" class="mx-auto text-dust-300 dark:text-pearl-400" />
                <p class="mt-3 text-sm text-secondary-fg">
                  {{ t('wallet.noActiveOrder') }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="hairline border-b px-6 py-4">
            <h2 class="text-lg font-semibold text-primary-fg">
              {{ t('wallet.orderHistory') }}
            </h2>
          </div>
          <div class="p-6">
            <div v-if="wallet.orders.length === 0" class="py-12 text-center">
              <Icon name="list" size="xl" class="mx-auto text-dust-300 dark:text-pearl-400" />
              <p class="mt-3 text-sm text-secondary-fg">
                {{ t('wallet.noOrders') }}
              </p>
            </div>

            <div v-else class="overflow-x-auto">
              <table class="w-full min-w-[760px]">
                <thead>
                  <tr class="hairline-strong border-b text-left text-sm text-secondary-fg">
                    <th class="px-3 py-3">{{ t('wallet.orderNo') }}</th>
                    <th class="px-3 py-3">{{ t('wallet.paymentMethod') }}</th>
                    <th class="px-3 py-3">{{ t('wallet.orderAmount') }}</th>
                    <th class="px-3 py-3">{{ t('wallet.creditAmount') }}</th>
                    <th class="px-3 py-3">{{ t('common.status') }}</th>
                    <th class="px-3 py-3">{{ t('wallet.createdAt') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-paper-200 dark:divide-ink-700">
                  <tr v-for="order in wallet.orders" :key="order.id" class="hover-tint">
                    <td class="px-3 py-3 font-mono text-xs text-dust-700 dark:text-pearl-200">
                      {{ order.client_sn }}
                    </td>
                    <td class="px-3 py-3 text-sm text-dust-700 dark:text-pearl-200">
                      {{ order.payway_name || orderPaywayLabel(order.payway) }}
                    </td>
                    <td class="px-3 py-3 text-sm num-mono text-dust-700 dark:text-pearl-200">
                      {{ formatCny(order.amount_yuan) }}
                    </td>
                    <td class="px-3 py-3 text-sm font-medium num-mono text-mint-600 dark:text-mint-500">
                      {{ formatUsd(order.credit_amount) }}
                    </td>
                    <td class="px-3 py-3">
                      <span :class="['badge', orderStatusClass(order.status)]">
                        {{ orderStatusLabel(order.status) }}
                      </span>
                    </td>
                    <td class="px-3 py-3 text-sm text-secondary-fg">
                      {{ formatDateTime(order.created_at) }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import type { PaygOrder, PaygWallet } from '@/types'
import { paygAPI } from '@/api/payg'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'

const PAYWAY_ALIPAY = '1'
const PAYWAY_WECHAT = '3'
const POLL_INTERVAL_MS = 3000

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const creatingOrder = ref(false)
const syncingOrder = ref(false)
const wallet = ref<PaygWallet | null>(null)
const selectedAmount = ref<number | null>(null)
const customAmount = ref('')
const payway = ref<string>(PAYWAY_ALIPAY)
const activeOrder = ref<PaygOrder | null>(null)
const activePaymentCode = ref('')
const activeQRCodeDataUrl = ref('')

let pollTimer: number | null = null

const fixedAmountOptions = computed(() => wallet.value?.fixed_amount_options ?? [])
const effectiveAmount = computed(() => {
  if (selectedAmount.value && selectedAmount.value > 0) {
    return selectedAmount.value
  }
  const amount = Number(customAmount.value)
  return Number.isFinite(amount) && amount > 0 ? amount : 0
})

const activePaywayClass =
  'rounded-xl border border-gold-500 dark:border-gold-300 bg-gold-500/10 dark:bg-gold-300/10 px-4 py-3 text-sm font-medium text-gold-600 dark:text-gold-300 transition'
const inactivePaywayClass =
  'rounded-xl border border-paper-200 dark:border-ink-700 px-4 py-3 text-sm font-medium text-dust-700 dark:text-pearl-100 transition hover:border-gold-500/50 dark:hover:border-gold-300/50 hover:text-gold-600 dark:hover:text-gold-300'

function formatUsd(value: number): string {
  return formatCurrency(value)
}

function formatCny(value: number): string {
  return formatCurrency(value, 'CNY')
}

function orderStatusLabel(status: string): string {
  switch (status) {
    case 'PAID':
      return t('wallet.statusPaid')
    case 'CLOSED':
      return t('wallet.statusClosed')
    default:
      return t('wallet.statusPending')
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
  return code === PAYWAY_WECHAT ? t('wallet.wechat') : t('wallet.alipay')
}

function stopPolling(): void {
  if (pollTimer !== null) {
    window.clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling(orderID: number): void {
  stopPolling()
  pollTimer = window.setInterval(() => {
    void syncOrder(orderID, true)
  }, POLL_INTERVAL_MS)
}

async function generateQRCode(paymentCode: string): Promise<void> {
  if (!paymentCode) {
    activeQRCodeDataUrl.value = ''
    return
  }
  activeQRCodeDataUrl.value = await QRCode.toDataURL(paymentCode, {
    width: 256,
    margin: 2,
    color: {
      dark: '#111827',
      light: '#ffffff',
    },
  })
}

async function loadWallet(): Promise<void> {
  loading.value = true
  try {
    wallet.value = await paygAPI.getWallet()
    if (!selectedAmount.value && fixedAmountOptions.value.length > 0) {
      selectedAmount.value = fixedAmountOptions.value[0]
    }
  } catch (error: any) {
    appStore.showError(
      t('wallet.loadFailed') + ': ' + (error.message || t('common.unknownError'))
    )
  } finally {
    loading.value = false
  }
}

function selectFixedAmount(amount: number): void {
  selectedAmount.value = amount
  customAmount.value = ''
}

async function createOrder(): Promise<void> {
  if (!wallet.value?.enabled) {
    return
  }
  if (effectiveAmount.value <= 0) {
    appStore.showError(t('wallet.invalidAmount'))
    return
  }

  creatingOrder.value = true
  try {
    const result = await paygAPI.precreate({
      amount: Number(effectiveAmount.value.toFixed(2)),
      payway: payway.value,
    })
    activeOrder.value = result.order
    activePaymentCode.value = result.qr_code
    await generateQRCode(result.qr_code)
    startPolling(result.order.id)
    await loadWallet()
  } catch (error: any) {
    appStore.showError(
      t('wallet.createOrderFailed') + ': ' + (error.message || t('common.unknownError'))
    )
  } finally {
    creatingOrder.value = false
  }
}

async function syncOrder(orderID: number, silent = false): Promise<void> {
  if (!silent) {
    syncingOrder.value = true
  }
  try {
    const previousStatus = activeOrder.value?.status
    const order = await paygAPI.queryOrder(orderID)
    activeOrder.value = order
    if (order.status !== 'PENDING') {
      stopPolling()
      await loadWallet()
    }
    if (previousStatus === 'PENDING' && order.status === 'PAID') {
      appStore.showSuccess(t('wallet.paymentSuccess'))
    }
  } catch (error: any) {
    if (!silent) {
      appStore.showError(
        t('wallet.syncOrderFailed') + ': ' + (error.message || t('common.unknownError'))
      )
    }
  } finally {
    if (!silent) {
      syncingOrder.value = false
    }
  }
}

async function refreshActiveOrder(): Promise<void> {
  if (!activeOrder.value) {
    return
  }
  syncingOrder.value = true
  try {
    await syncOrder(activeOrder.value.id)
  } finally {
    syncingOrder.value = false
  }
}

async function copyPaymentCode(): Promise<void> {
  if (!activePaymentCode.value) {
    return
  }
  await copyToClipboard(activePaymentCode.value, t('wallet.paymentCodeCopied'))
}

onMounted(() => {
  void loadWallet()
})

onUnmounted(() => {
  stopPolling()
})
</script>
