<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <!-- Page Header -->
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('nav.referral') }}
        </h1>
        <p class="mt-1 text-gray-500 dark:text-gray-400">
          {{ t('profile.referral.description') }}
        </p>
      </div>

      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center py-16">
        <div class="animate-spin rounded-full h-10 w-10 border-b-2 border-primary-500"></div>
      </div>

      <!-- System disabled -->
      <div v-else-if="!settings?.enabled" class="card p-8">
        <div class="flex items-center gap-4">
          <div class="flex-shrink-0 rounded-full bg-gray-100 p-4 dark:bg-dark-700">
            <Icon name="userPlus" size="xl" class="text-gray-400" />
          </div>
          <div>
            <p class="text-lg font-medium text-gray-700 dark:text-gray-300">
              {{ t('profile.referral.systemDisabled') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Referral enabled -->
      <template v-else>
        <!-- Reward Rules Card -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.rewardRules') }}
            </h2>
          </div>
          <div class="p-6">
            <!-- Two reward cards -->
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <!-- Referrer register bonus -->
              <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
                <div class="flex items-center justify-between">
                  <span class="font-medium text-gray-900 dark:text-white">{{ t('referral.referrerRegister') }}</span>
                  <span class="rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">
                    {{ t('referral.basicReward') }}
                  </span>
                </div>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('referral.referrerRegisterDesc') }}</p>
                <p class="mt-2 text-2xl font-bold text-green-600 dark:text-green-400">+${{ settings?.register_bonus?.toFixed(2) || '0.00' }}</p>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('referral.perInvite') }}</p>
              </div>

              <!-- Referrer commission -->
              <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
                <div class="flex items-center justify-between">
                  <span class="font-medium text-gray-900 dark:text-white">{{ t('referral.referrerCommission') }}</span>
                  <span class="rounded-full bg-orange-100 px-2.5 py-0.5 text-xs font-medium text-orange-700 dark:bg-orange-900/30 dark:text-orange-400">
                    {{ t('referral.advancedRebate') }}
                  </span>
                </div>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('referral.referrerCommissionDesc') }}</p>
                <p class="mt-2 text-2xl font-bold text-orange-600 dark:text-orange-400">{{ ((referralInfo?.commission_rate || settings?.commission_rate || 0) * 100).toFixed(0) }}%</p>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('referral.ofPurchase') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- My Referral Link Card -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.myLink') }}
            </h2>
          </div>
          <div class="p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <div class="flex-1 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 font-mono text-sm dark:border-dark-600 dark:bg-dark-800 overflow-x-auto">
                {{ computedReferralLink || '-' }}
              </div>
              <div class="flex gap-2">
                <button
                  type="button"
                  class="btn btn-primary flex-1 sm:flex-none"
                  @click="copyLink"
                >
                  <Icon name="copy" size="sm" class="mr-1" />
                  {{ t('referral.copyLink') }}
                </button>
                <button
                  type="button"
                  class="btn btn-outline"
                  :title="t('referral.showQRCode')"
                  @click="showQRCode"
                >
                  <Icon name="qrCode" size="md" />
                </button>
                <button
                  type="button"
                  class="btn btn-outline"
                  :title="t('referral.copyCode')"
                  @click="copyCode"
                >
                  <Icon name="key" size="md" />
                </button>
              </div>
            </div>
            <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
              {{ t('referral.linkHint') }}
            </p>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('profile.referral.myCode') }}: <span class="font-mono font-medium text-gray-700 dark:text-gray-300">{{ referralInfo?.referral_code || '-' }}</span>
            </p>
          </div>
        </div>

        <!-- Statistics Card -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.statistics') }}
            </h2>
          </div>
          <div class="p-6">
            <div class="grid grid-cols-2 gap-4 md:grid-cols-4">
              <!-- Total Reward -->
              <div class="rounded-lg border border-gray-200 p-4 text-center dark:border-dark-600">
                <p class="text-2xl font-bold text-green-600 dark:text-green-400">
                  ${{ (referralInfo?.total_reward || 0).toFixed(2) }}
                </p>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('referral.totalEarned') }}</p>
              </div>

              <!-- Total Invited -->
              <div class="rounded-lg border border-gray-200 p-4 text-center dark:border-dark-600">
                <p class="text-2xl font-bold text-gray-900 dark:text-white">
                  {{ referralInfo?.total_invited || 0 }}
                </p>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('referral.totalInvited') }}</p>
              </div>

              <!-- Register Reward -->
              <div class="rounded-lg border border-gray-200 p-4 text-center dark:border-dark-600">
                <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">
                  ${{ (referralInfo?.register_reward || 0).toFixed(2) }}
                </p>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('referral.registerReward') }}</p>
              </div>

              <!-- Commission Reward -->
              <div class="rounded-lg border border-gray-200 p-4 text-center dark:border-dark-600">
                <p class="text-2xl font-bold text-orange-600 dark:text-orange-400">
                  ${{ (referralInfo?.commission_reward || 0).toFixed(2) }}
                </p>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('referral.commissionReward') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Referral Records Card -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.records') }}
            </h2>
          </div>
          <div class="p-6">
            <!-- Loading -->
            <div v-if="recordsLoading" class="flex items-center justify-center py-8">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
            </div>

            <!-- Empty state -->
            <div v-else-if="rewards.length === 0" class="py-8 text-center">
              <Icon name="gift" size="xl" class="mx-auto text-gray-300 dark:text-dark-600" />
              <p class="mt-2 text-gray-500 dark:text-dark-400">{{ t('profile.referral.noRewards') }}</p>
            </div>

            <!-- Records table -->
            <div v-else class="overflow-x-auto">
              <table class="w-full">
                <thead>
                  <tr class="border-b border-gray-200 dark:border-dark-700">
                    <th class="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-dark-400">
                      {{ t('referral.referredUser') }}
                    </th>
                    <th class="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-dark-400">
                      {{ t('referral.time') }}
                    </th>
                    <th class="px-4 py-3 text-center text-sm font-medium text-gray-500 dark:text-dark-400">
                      {{ t('referral.rewardType') }}
                    </th>
                    <th class="px-4 py-3 text-right text-sm font-medium text-gray-500 dark:text-dark-400">
                      {{ t('referral.rewardAmount') }}
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr
                    v-for="reward in rewards"
                    :key="reward.id"
                    class="hover:bg-gray-50 dark:hover:bg-dark-700/50"
                  >
                    <td class="px-4 py-3">
                      <p class="text-sm text-gray-900 dark:text-white">{{ maskEmail(reward.referee_email) }}</p>
                    </td>
                    <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">
                      {{ formatDate(reward.created_at) }}
                    </td>
                    <td class="px-4 py-3 text-center">
                      <span
                        :class="[
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                          reward.reward_type === 'register'
                            ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                            : 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                        ]"
                      >
                        {{ reward.reward_type === 'register' ? t('referral.registerType') : t('referral.commissionType') }}
                      </span>
                    </td>
                    <td class="px-4 py-3 text-right">
                      <span class="text-sm font-medium text-green-600 dark:text-green-400">
                        +${{ reward.reward_amount.toFixed(2) }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>

              <!-- Pagination -->
              <div v-if="totalPages > 1" class="mt-4 flex items-center justify-between border-t border-gray-200 pt-4 dark:border-dark-700">
                <div class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('pagination.pageOf', { page, total: totalPages }) }}
                </div>
                <nav class="flex items-center gap-1">
                  <button
                    type="button"
                    class="px-3 py-1.5 rounded text-sm border border-gray-200 dark:border-dark-600"
                    :disabled="page === 1"
                    :class="page === 1 ? 'text-gray-300 dark:text-dark-600 cursor-not-allowed' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700'"
                    @click="loadRecords(page - 1)"
                  >
                    {{ t('pagination.previous') }}
                  </button>
                  <button
                    type="button"
                    class="px-3 py-1.5 rounded text-sm border border-gray-200 dark:border-dark-600"
                    :disabled="page === totalPages"
                    :class="page === totalPages ? 'text-gray-300 dark:text-dark-600 cursor-not-allowed' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700'"
                    @click="loadRecords(page + 1)"
                  >
                    {{ t('pagination.next') }}
                  </button>
                </nav>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- QR Code Modal -->
      <Teleport to="body">
        <div
          v-if="showQRModal"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          @click.self="showQRModal = false"
        >
          <div class="mx-4 w-full max-w-sm rounded-lg bg-white p-6 shadow-xl dark:bg-dark-800">
            <div class="flex items-center justify-between mb-4">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('referral.qrCodeTitle') }}
              </h3>
              <button
                type="button"
                class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                @click="showQRModal = false"
              >
                <Icon name="x" size="md" />
              </button>
            </div>
            <div class="flex flex-col items-center">
              <div class="rounded-lg bg-white p-2">
                <img
                  v-if="qrCodeDataUrl"
                  :src="qrCodeDataUrl"
                  alt="Referral QR Code"
                  class="h-64 w-64"
                />
              </div>
              <p class="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('referral.qrCodeHint') }}
              </p>
            </div>
          </div>
        </div>
      </Teleport>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { referralAPI } from '@/api/referral'
import { copyToClipboard } from '@/utils/clipboard'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import QRCode from 'qrcode'
import type { ReferralInfo, ReferralSettings, ReferralReward } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const recordsLoading = ref(false)
const referralInfo = ref<ReferralInfo | null>(null)
const settings = ref<ReferralSettings | null>(null)
const rewards = ref<ReferralReward[]>([])
const page = ref(1)
const pageSize = 10
const totalPages = ref(1)

// QR code modal state
const showQRModal = ref(false)
const qrCodeDataUrl = ref('')

// Computed referral link - use backend value or generate from code
const computedReferralLink = computed(() => {
  if (referralInfo.value?.referral_link) {
    return referralInfo.value.referral_link
  }
  const code = referralInfo.value?.referral_code
  if (code) {
    return `${window.location.origin}/register?ref=${encodeURIComponent(code)}`
  }
  return ''
})

async function loadData(): Promise<void> {
  loading.value = true
  try {
    const [infoRes, settingsRes] = await Promise.all([
      referralAPI.getReferralInfo(),
      referralAPI.getReferralSettings()
    ])
    referralInfo.value = infoRes
    settings.value = settingsRes
    // Load records after main data
    if (settingsRes.enabled) {
      await loadRecords(1)
    }
  } catch (error) {
    console.error('Failed to load referral data:', error)
  } finally {
    loading.value = false
  }
}

async function loadRecords(newPage: number): Promise<void> {
  if (newPage < 1) return
  recordsLoading.value = true
  try {
    const res = await referralAPI.getReferralRewards(newPage, pageSize)
    rewards.value = res.items
    totalPages.value = Math.ceil(res.total / pageSize) || 1
    page.value = newPage
  } catch (error) {
    console.error('Failed to load rewards:', error)
  } finally {
    recordsLoading.value = false
  }
}

async function copyLink(): Promise<void> {
  const link = computedReferralLink.value
  if (!link) return
  const success = await copyToClipboard(link)
  if (success) {
    appStore.showSuccess(t('profile.referral.linkCopied'))
  } else {
    appStore.showError(t('common.copyFailed'))
  }
}

async function copyCode(): Promise<void> {
  const code = referralInfo.value?.referral_code
  if (!code) return
  const success = await copyToClipboard(code)
  if (success) {
    appStore.showSuccess(t('profile.referral.codeCopied'))
  } else {
    appStore.showError(t('common.copyFailed'))
  }
}

async function showQRCode(): Promise<void> {
  const link = computedReferralLink.value
  if (!link) return
  try {
    qrCodeDataUrl.value = await QRCode.toDataURL(link, {
      width: 256,
      margin: 2,
      color: {
        dark: '#000000',
        light: '#ffffff'
      }
    })
    showQRModal.value = true
  } catch (err) {
    console.error('Failed to generate QR code:', err)
    appStore.showError(t('common.error'))
  }
}

function maskEmail(email: string): string {
  if (!email) return '-'
  const [local, domain] = email.split('@')
  if (!domain) return email
  const maskedLocal = local.length > 3 ? local.slice(0, 3) + '***' : local + '***'
  return `${maskedLocal}@${domain}`
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(loadData)
</script>
