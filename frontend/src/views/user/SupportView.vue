<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('support.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('support.description') }}
        </p>
      </div>

      <template v-if="contactInfo">
        <div class="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_320px]">
          <UserDashboardContact />

          <div
            class="card border-blue-100 bg-gradient-to-br from-blue-50 via-white to-cyan-50 dark:border-blue-900/50 dark:from-blue-900/20 dark:via-dark-800 dark:to-cyan-900/10"
          >
            <div class="border-b border-blue-100 px-6 py-4 dark:border-blue-900/40">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('support.quickCopyTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('support.quickCopyDescription') }}
              </p>
            </div>
            <div class="space-y-4 p-6">
              <div class="rounded-xl border border-blue-100 bg-white/80 p-4 text-sm text-gray-600 dark:border-blue-900/40 dark:bg-dark-800/80 dark:text-gray-300">
                <div class="font-medium text-gray-900 dark:text-white">
                  {{ t('support.contactBlockTitle') }}
                </div>
                <pre class="mt-3 whitespace-pre-wrap break-words font-sans text-sm leading-6 text-gray-600 dark:text-gray-300">{{ contactInfo }}</pre>
              </div>

              <button type="button" class="btn btn-primary w-full" @click="copyAllContactInfo">
                <Icon name="copy" size="sm" class="mr-1.5" />
                {{ t('support.copyAll') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <div
        v-else
        class="card border-amber-200 bg-amber-50/80 p-6 text-sm text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/10 dark:text-amber-300"
      >
        <div class="flex items-start gap-3">
          <Icon name="exclamationTriangle" size="md" class="mt-0.5 text-amber-500" />
          <div>
            <div class="font-medium">{{ t('support.notConfiguredTitle') }}</div>
            <div class="mt-1">{{ t('support.notConfiguredDescription') }}</div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import UserDashboardContact from '@/components/user/dashboard/UserDashboardContact.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const contactInfo = computed(() => appStore.cachedPublicSettings?.contact_info || appStore.contactInfo || '')

async function copyAllContactInfo(): Promise<void> {
  await copyToClipboard(contactInfo.value, t('support.copyAllSuccess'))
}
</script>
