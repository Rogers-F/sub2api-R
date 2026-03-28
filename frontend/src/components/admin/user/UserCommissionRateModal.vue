<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.commissionRate.title')"
    width="narrow"
    @close="emit('close')"
  >
    <div v-if="user" class="space-y-5">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-accent-100 dark:bg-accent-800/30">
          <span class="text-lg font-medium text-accent-700 dark:text-accent-300">
            {{ user.email.charAt(0).toUpperCase() }}
          </span>
        </div>
        <div class="flex-1">
          <p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.users.commissionRate.description', { email: user.email }) }}
          </p>
        </div>
      </div>

      <div v-if="loading" class="flex justify-center py-10">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>

      <template v-else>
        <div class="grid gap-3 sm:grid-cols-3">
          <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.users.commissionRate.globalRate') }}
            </p>
            <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ formatPercent(info?.global_commission_rate ?? null) }}
            </p>
          </div>

          <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.users.commissionRate.userRate') }}
            </p>
            <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ info?.user_commission_rate == null ? t('admin.users.commissionRate.inherited') : formatPercent(info.user_commission_rate) }}
            </p>
          </div>

          <div class="rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-800/60 dark:bg-primary-900/20">
            <p class="text-sm text-primary-700 dark:text-primary-300">
              {{ t('admin.users.commissionRate.effectiveRate') }}
            </p>
            <p class="mt-2 text-lg font-semibold text-primary-700 dark:text-primary-200">
              {{ formatPercent(info?.effective_rate ?? null) }}
            </p>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-600">
          <div class="grid gap-2 sm:grid-cols-2">
            <button
              type="button"
              class="rounded-lg border px-3 py-2 text-sm font-medium transition-colors"
              :class="useGlobalRate
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
                : 'border-gray-200 text-gray-600 hover:border-gray-300 dark:border-dark-600 dark:text-gray-300'"
              @click="useGlobalRate = true"
            >
              {{ t('admin.users.commissionRate.useGlobal') }}
            </button>
            <button
              type="button"
              class="rounded-lg border px-3 py-2 text-sm font-medium transition-colors"
              :class="!useGlobalRate
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
                : 'border-gray-200 text-gray-600 hover:border-gray-300 dark:border-dark-600 dark:text-gray-300'"
              @click="useGlobalRate = false"
            >
              {{ t('admin.users.commissionRate.useCustom') }}
            </button>
          </div>

          <div v-if="!useGlobalRate" class="mt-4">
            <label class="input-label">{{ t('admin.users.commissionRate.customRateLabel') }}</label>
            <div class="relative">
              <input
                v-model="customRatePercent"
                type="number"
                min="0"
                max="100"
                step="0.01"
                class="input pr-8"
                :placeholder="t('admin.users.commissionRate.customRatePlaceholder')"
              />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">%</span>
            </div>
            <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.users.commissionRate.customRateHint') }}
            </p>
          </div>
        </div>
      </template>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="loading || submitting"
          @click="handleSave"
        >
          {{ submitting ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminUser, UserCommissionRateInfo } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{
  show: boolean
  user: AdminUser | null
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const info = ref<UserCommissionRateInfo | null>(null)
const useGlobalRate = ref(true)
const customRatePercent = ref('')

watch(
  [() => props.show, () => props.user?.id],
  ([show, userId]) => {
    if (show && userId) {
      void loadCommissionRate(userId)
    }
  }
)

const formatPercent = (value: number | null | undefined) => {
  if (value == null || Number.isNaN(value)) {
    return '--'
  }
  return `${(value * 100).toFixed(2).replace(/\.?0+$/, '')}%`
}

const formatPercentInput = (value: number) => (value * 100).toFixed(2).replace(/\.?0+$/, '')

const loadCommissionRate = async (userId: number) => {
  loading.value = true
  try {
    const response = await adminAPI.users.getCommissionRate(userId)
    info.value = response
    useGlobalRate.value = response.user_commission_rate == null
    customRatePercent.value =
      response.user_commission_rate == null ? '' : formatPercentInput(response.user_commission_rate)
  } catch (error: any) {
    console.error('Failed to load commission rate:', error)
    appStore.showError(error.message || t('admin.users.commissionRate.loadFailed'))
  } finally {
    loading.value = false
  }
}

const parseCustomRate = () => {
  if (customRatePercent.value.trim() === '') {
    appStore.showError(t('admin.users.commissionRate.invalidRate'))
    return null
  }
  const value = Number(customRatePercent.value)
  if (!Number.isFinite(value) || value < 0 || value > 100) {
    appStore.showError(t('admin.users.commissionRate.invalidRate'))
    return null
  }
  return value / 100
}

const handleSave = async () => {
  if (!props.user) {
    return
  }

  let nextRate: number | null = null
  if (!useGlobalRate.value) {
    nextRate = parseCustomRate()
    if (nextRate == null) {
      return
    }
  }

  submitting.value = true
  try {
    const response = await adminAPI.users.updateCommissionRate(props.user.id, nextRate)
    info.value = response
    appStore.showSuccess(t('admin.users.commissionRate.saveSuccess'))
    emit('success')
    emit('close')
  } catch (error: any) {
    console.error('Failed to update commission rate:', error)
    appStore.showError(error.message || t('admin.users.commissionRate.saveFailed'))
  } finally {
    submitting.value = false
  }
}
</script>
