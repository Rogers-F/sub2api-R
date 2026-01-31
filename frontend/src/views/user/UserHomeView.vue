<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Page Title -->
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('nav.userHome') }}
        </h1>
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
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import UserDashboardQuickStart from '@/components/user/dashboard/UserDashboardQuickStart.vue'
import UserDashboardContact from '@/components/user/dashboard/UserDashboardContact.vue'
import UserDashboardAnnouncements from '@/components/user/dashboard/UserDashboardAnnouncements.vue'

const { t } = useI18n()
</script>
