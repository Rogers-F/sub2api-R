<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Welcome Section + Stats -->
      <div>
        <div class="mb-4">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('dashboard.welcomeBack') }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ authStore.user?.username }}</p>
        </div>

        <!-- Stats Cards -->
        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <!-- API Keys -->
          <div class="card p-4">
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.apiKeys') }}</p>
              <Icon name="key" size="md" class="text-gray-400" />
            </div>
            <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ stats?.total_api_keys || 0 }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('common.active') }}: {{ stats?.active_api_keys || 0 }}</p>
          </div>

          <!-- Subscriptions -->
          <div class="card p-4">
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.subscriptionStatus') }}</p>
              <Icon name="calendar" size="md" class="text-gray-400" />
            </div>
            <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ subscriptionStore.activeSubscriptions.length }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.activeSubscriptions') }}</p>
          </div>

          <!-- Today Tokens -->
          <div class="card p-4">
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.todayTokens') }}</p>
              <Icon name="chart" size="md" class="text-gray-400" />
            </div>
            <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ formatTokens(stats?.today_tokens || 0) }}</p>
            <p class="text-xs text-emerald-600 dark:text-emerald-400">{{ t('dashboard.todayCost') }}: ${{ formatCost(stats?.today_actual_cost || 0) }}</p>
          </div>

          <!-- Total Tokens -->
          <div class="card p-4">
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.totalTokens') }}</p>
              <Icon name="database" size="md" class="text-gray-400" />
            </div>
            <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ formatTokens(stats?.total_tokens || 0) }}</p>
            <p class="text-xs text-emerald-600 dark:text-emerald-400">{{ t('dashboard.totalCost') }}: ${{ formatCost(stats?.total_actual_cost || 0) }}</p>
          </div>
        </div>
      </div>

      <!-- Quick Start + Announcements Row -->
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div class="space-y-6">
          <UserDashboardQuickStart />
          <UserDashboardContact />
        </div>
        <div>
          <UserDashboardAnnouncements />
        </div>
      </div>

      <!-- Quick Actions (4 columns) -->
      <UserDashboardQuickActions />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import UserDashboardQuickStart from '@/components/user/dashboard/UserDashboardQuickStart.vue'
import UserDashboardContact from '@/components/user/dashboard/UserDashboardContact.vue'
import UserDashboardAnnouncements from '@/components/user/dashboard/UserDashboardAnnouncements.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore, useSubscriptionStore } from '@/stores'
import { getDashboardStats, type UserDashboardStats } from '@/api/usage'

const { t } = useI18n()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()

const stats = ref<UserDashboardStats | null>(null)

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toString()
}

function formatCost(c: number): string {
  return c.toFixed(4)
}

onMounted(async () => {
  try {
    stats.value = await getDashboardStats()
  } catch (e) {
    console.error('Failed to load dashboard stats:', e)
  }
  // Ensure subscriptions are loaded
  subscriptionStore.fetchActiveSubscriptions()
})
</script>
