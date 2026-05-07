<template>
  <div class="card">
    <div class="border-b hairline px-6 py-4">
      <h2 class="text-lg font-semibold text-dust-900 dark:text-white">{{ t('dashboard.quickActions') }}</h2>
    </div>
    <div class="grid grid-cols-2 gap-3 p-4 lg:grid-cols-4">
      <!-- API Keys -->
      <button
        @click="router.push('/keys')"
        class="group flex flex-col items-center gap-2 rounded-xl bg-paper-50 p-4 text-center transition-all duration-200 hover:bg-paper-100 dark:bg-ink-800/50 dark:hover:bg-ink-800"
      >
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-gold-500/15 transition-transform group-hover:scale-105 dark:bg-gold-300/15">
          <Icon name="key" size="lg" class="text-gold-600 dark:text-gold-400" />
        </div>
        <div>
          <p class="text-sm font-medium text-dust-900 dark:text-white">{{ t('nav.apiKeys') }}</p>
          <p class="text-xs text-dust-500 dark:text-pearl-300">{{ t('dashboard.manageKeys') }}</p>
        </div>
      </button>

      <!-- Subscriptions -->
      <button
        @click="router.push('/subscriptions')"
        class="group flex flex-col items-center gap-2 rounded-xl bg-paper-50 p-4 text-center transition-all duration-200 hover:bg-paper-100 dark:bg-ink-800/50 dark:hover:bg-ink-800"
      >
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-mint-500/15 transition-transform group-hover:scale-105 dark:bg-mint-500/20">
          <Icon name="checkCircle" size="lg" class="text-mint-600 dark:text-mint-400" />
        </div>
        <div>
          <p class="text-sm font-medium text-dust-900 dark:text-white">{{ t('nav.subscriptions') }}</p>
          <p class="text-xs text-dust-500 dark:text-pearl-300">{{ t('dashboard.viewSubscriptions') }}</p>
        </div>
      </button>

      <!-- Usage Statistics -->
      <button
        @click="router.push('/usage')"
        class="group flex flex-col items-center gap-2 rounded-xl bg-paper-50 p-4 text-center transition-all duration-200 hover:bg-paper-100 dark:bg-ink-800/50 dark:hover:bg-ink-800"
      >
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-100 transition-transform group-hover:scale-105 dark:bg-blue-900/30">
          <Icon name="chart" size="lg" class="text-blue-600 dark:text-blue-400" />
        </div>
        <div>
          <p class="text-sm font-medium text-dust-900 dark:text-white">{{ t('nav.usageStatistics') }}</p>
          <p class="text-xs text-dust-500 dark:text-pearl-300">{{ t('dashboard.checkUsage') }}</p>
        </div>
      </button>

      <!-- Redeem Code -->
      <button
        @click="router.push('/redeem')"
        class="group flex flex-col items-center gap-2 rounded-xl bg-paper-50 p-4 text-center transition-all duration-200 hover:bg-paper-100 dark:bg-ink-800/50 dark:hover:bg-ink-800"
      >
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-100 transition-transform group-hover:scale-105 dark:bg-amber-900/30">
          <Icon name="gift" size="lg" class="text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <p class="text-sm font-medium text-dust-900 dark:text-white">{{ t('nav.redeemCode') }}</p>
          <p class="text-xs text-dust-500 dark:text-pearl-300">{{ t('dashboard.redeemNow') }}</p>
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const router = useRouter()
const { t } = useI18n()
</script>
