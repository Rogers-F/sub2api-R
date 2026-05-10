<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-gold-500 dark:border-gold-300 border-t-transparent"
        ></div>
      </div>

      <!-- Empty State -->
      <div v-else-if="subscriptions.length === 0" class="card p-12 text-center">
        <div
          class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-paper-100 dark:bg-ink-700"
        >
          <Icon name="creditCard" size="xl" class="text-dust-400" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-primary-fg">
          {{ t('userSubscriptions.noActiveSubscriptions') }}
        </h3>
        <p class="text-secondary-fg">
          {{ t('userSubscriptions.noActiveSubscriptionsDesc') }}
        </p>
      </div>

      <!-- Subscriptions Grid -->
      <div v-else class="grid gap-6 lg:grid-cols-2">
        <div
          v-for="subscription in subscriptions"
          :key="subscription.id"
          class="card overflow-hidden"
        >
          <!-- Header -->
          <div
            class="flex items-center justify-between hairline border-b p-4"
          >
            <div class="flex items-center gap-3">
              <div
                class="flex h-10 w-10 items-center justify-center rounded-xl bg-purple-100 dark:bg-purple-900/30"
              >
                <Icon name="creditCard" size="md" class="text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <h3 class="font-semibold text-primary-fg">
                  {{ subscription.group?.name || `Group #${subscription.group_id}` }}
                </h3>
                <p class="text-xs text-secondary-fg">
                  {{ subscription.group?.description || '' }}
                </p>
              </div>
            </div>
            <span
              :class="[
                'badge',
                subscription.status === 'active'
                  ? 'badge-success'
                  : subscription.status === 'expired'
                    ? 'badge-warning'
                    : 'badge-danger'
              ]"
            >
              {{ t(`userSubscriptions.status.${subscription.status}`) }}
            </span>
          </div>

          <!-- Usage Progress -->
          <div class="space-y-4 p-4">
            <!-- Expiration Info -->
            <div v-if="subscription.expires_at" class="flex items-center justify-between text-sm">
              <span class="text-secondary-fg">{{
                t('userSubscriptions.expires')
              }}</span>
              <span :class="getExpirationClass(subscription.expires_at)">
                {{ formatExpirationDate(subscription.expires_at) }}
              </span>
            </div>
            <div v-else class="flex items-center justify-between text-sm">
              <span class="text-secondary-fg">{{
                t('userSubscriptions.expires')
              }}</span>
              <span class="text-dust-700 dark:text-pearl-100">{{
                t('userSubscriptions.noExpiration')
              }}</span>
            </div>

            <!-- Daily Usage -->
            <div v-if="subscription.group?.daily_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-dust-700 dark:text-pearl-100">
                  {{ t('userSubscriptions.daily') }}
                </span>
                <span class="text-sm text-secondary-fg">
                  ${{ (subscription.daily_usage_usd || 0).toFixed(2) }} / ${{
                    subscription.group.daily_limit_usd.toFixed(2)
                  }}
                </span>
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-paper-200 dark:bg-ink-700">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.daily_usage_usd,
                      subscription.group.daily_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.daily_usage_usd,
                      subscription.group.daily_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="formatWindowStatus(subscription.daily_reset_status)"
                class="text-xs text-secondary-fg whitespace-pre-line"
                :class="getResetStatusClass(subscription.daily_reset_status)"
              >
                {{ formatWindowStatus(subscription.daily_reset_status) }}
              </p>
            </div>

            <!-- Weekly Usage -->
            <div v-if="subscription.group?.weekly_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-dust-700 dark:text-pearl-100">
                  {{ t('userSubscriptions.weekly') }}
                </span>
                <span class="text-sm text-secondary-fg">
                  ${{ (subscription.weekly_usage_usd || 0).toFixed(2) }} / ${{
                    subscription.group.weekly_limit_usd.toFixed(2)
                  }}
                </span>
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-paper-200 dark:bg-ink-700">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.weekly_usage_usd,
                      subscription.group.weekly_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.weekly_usage_usd,
                      subscription.group.weekly_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="formatWindowStatus(subscription.weekly_reset_status)"
                class="text-xs text-secondary-fg whitespace-pre-line"
                :class="getResetStatusClass(subscription.weekly_reset_status)"
              >
                {{ formatWindowStatus(subscription.weekly_reset_status) }}
              </p>
            </div>

            <!-- Monthly Usage -->
            <div v-if="subscription.group?.monthly_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-dust-700 dark:text-pearl-100">
                  {{ t('userSubscriptions.monthly') }}
                </span>
                <span class="text-sm text-secondary-fg">
                  ${{ (subscription.monthly_usage_usd || 0).toFixed(2) }} / ${{
                    subscription.group.monthly_limit_usd.toFixed(2)
                  }}
                </span>
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-paper-200 dark:bg-ink-700">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.monthly_usage_usd,
                      subscription.group.monthly_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.monthly_usage_usd,
                      subscription.group.monthly_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="formatWindowStatus(subscription.monthly_reset_status)"
                class="text-xs text-secondary-fg whitespace-pre-line"
                :class="getResetStatusClass(subscription.monthly_reset_status)"
              >
                {{ formatWindowStatus(subscription.monthly_reset_status) }}
              </p>
            </div>

            <!-- No limits configured - Unlimited badge -->
            <div
              v-if="
                !subscription.group?.daily_limit_usd &&
                !subscription.group?.weekly_limit_usd &&
                !subscription.group?.monthly_limit_usd
              "
              class="flex items-center justify-center rounded-xl bg-mint-500/10 py-6 dark:bg-mint-500/10"
            >
              <div class="flex items-center gap-3">
                <span class="text-4xl text-mint-600 dark:text-mint-500">∞</span>
                <div>
                  <p class="text-sm font-medium text-mint-600 dark:text-mint-500">
                    {{ t('userSubscriptions.unlimited') }}
                  </p>
                  <p class="text-xs text-mint-600/70 dark:text-mint-500/70">
                    {{ t('userSubscriptions.unlimitedDesc') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import subscriptionsAPI from '@/api/subscriptions'
import type { UserSubscription } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useWindowResetStatus } from '@/composables/useWindowResetStatus'

const { t } = useI18n()
const appStore = useAppStore()
const { formatWindowStatus, getResetStatusClass } = useWindowResetStatus('userSubscriptions.resetStatus')

const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)

async function loadSubscriptions() {
  try {
    loading.value = true
    subscriptions.value = await subscriptionsAPI.getMySubscriptions()
  } catch (error) {
    console.error('Failed to load subscriptions:', error)
    appStore.showError(t('userSubscriptions.failedToLoad'))
  } finally {
    loading.value = false
  }
}

function getProgressWidth(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return '0%'
  const percentage = Math.min(((used || 0) / limit) * 100, 100)
  return `${percentage}%`
}

function getProgressBarClass(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return 'bg-dust-400'
  const percentage = ((used || 0) / limit) * 100
  if (percentage >= 90) return 'bg-coral-500'
  if (percentage >= 70) return 'bg-orange-500'
  return 'bg-green-500'
}

function formatExpirationDate(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()

  if (diff < 0) {
    return t('userSubscriptions.status.expired')
  }

  const month = expires.getMonth() + 1
  const day = expires.getDate()
  const hours = String(expires.getHours()).padStart(2, '0')
  const minutes = String(expires.getMinutes()).padStart(2, '0')
  const dateStr = `${expires.getFullYear()}/${String(month).padStart(2, '0')}/${String(day).padStart(2, '0')} ${hours}:${minutes}`

  const totalMinutes = Math.floor(diff / (1000 * 60))
  const d = Math.floor(totalMinutes / (60 * 24))
  const h = Math.floor((totalMinutes % (60 * 24)) / 60)
  const m = totalMinutes % 60

  let remaining: string
  if (d > 0) {
    remaining = `${d}d ${h}h`
  } else if (h > 0) {
    remaining = `${h}h ${m}m`
  } else {
    remaining = `${m}m`
  }

  return `${t('userSubscriptions.daysRemaining', { days: remaining })} (${dateStr})`
}

function getExpirationClass(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diffHours = (expires.getTime() - now.getTime()) / (1000 * 60 * 60)

  if (diffHours <= 0) return 'text-coral-600 dark:text-coral-500 font-medium'
  if (diffHours <= 72) return 'text-coral-600 dark:text-coral-500'
  if (diffHours <= 168) return 'text-orange-600 dark:text-orange-400'
  return 'text-dust-700 dark:text-pearl-100'
}

onMounted(() => {
  loadSubscriptions()
})
</script>
