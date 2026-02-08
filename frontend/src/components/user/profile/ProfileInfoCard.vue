<template>
  <div class="card overflow-hidden">
    <div
      class="border-b border-gray-100 bg-gradient-to-r from-accent-100 to-accent-50 px-6 py-5 dark:border-dark-700 dark:from-accent-800/60 dark:to-accent-900/40"
    >
      <div class="flex items-center gap-4">
        <!-- Avatar -->
        <div
          class="flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-accent-700 to-accent-900 text-2xl font-bold text-white shadow-lg shadow-accent-900/30"
        >
          {{ user?.email?.charAt(0).toUpperCase() || 'U' }}
        </div>
        <div class="min-w-0 flex-1">
          <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-white">
            {{ user?.email }}
          </h2>
          <div class="mt-1 flex items-center gap-2">
            <span :class="['badge', user?.role === 'admin' ? 'badge-primary' : 'badge-gray']">
              {{ user?.role === 'admin' ? t('profile.administrator') : t('profile.user') }}
            </span>
            <span
              :class="['badge', user?.status === 'active' ? 'badge-success' : 'badge-danger']"
            >
              {{ user?.status }}
            </span>
          </div>
        </div>
      </div>
    </div>
    <div class="px-6 py-4">
      <div class="space-y-3">
        <div class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
          <Icon name="mail" size="sm" class="text-gray-400 dark:text-gray-500" />
          <span class="truncate">{{ user?.email }}</span>
        </div>
        <div
          v-if="user?.username"
          class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400"
        >
          <Icon name="user" size="sm" class="text-gray-400 dark:text-gray-500" />
          <span class="truncate">{{ user.username }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { User } from '@/types'

defineProps<{
  user: User | null
}>()

const { t } = useI18n()
</script>
