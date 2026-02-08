<template>
  <BaseDialog :show="show" :title="t('admin.users.commissionRate.title')" width="narrow" @close="$emit('close')">
    <form v-if="user" id="commission-rate-form" @submit.prevent="handleSubmit" class="space-y-5">
      <!-- User Info -->
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-accent-100 dark:bg-accent-800/30">
          <span class="text-lg font-medium text-accent-700 dark:text-accent-300">{{ user.email.charAt(0).toUpperCase() }}</span>
        </div>
        <div class="flex-1">
          <p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-gray-400">ID: {{ user.id }}</p>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else>
        <!-- Global Rate Display -->
        <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700">
          <label class="input-label text-gray-500 dark:text-gray-400">{{ t('admin.users.commissionRate.globalRate') }}</label>
          <p class="text-lg font-medium text-gray-900 dark:text-white">{{ formatPercent(rateInfo.global_commission_rate) }}%</p>
        </div>

        <!-- Use Global Setting Checkbox -->
        <div class="flex items-center gap-3">
          <input
            id="use-global"
            v-model="useGlobalRate"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label for="use-global" class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.users.commissionRate.useGlobal') }}
          </label>
        </div>

        <!-- Custom Rate Input -->
        <div v-if="!useGlobalRate">
          <label class="input-label">{{ t('admin.users.commissionRate.customRate') }}</label>
          <div class="relative">
            <input
              v-model.number="customRatePercent"
              type="number"
              step="0.01"
              min="0"
              max="100"
              required
              class="input pr-8"
              :placeholder="t('admin.users.commissionRate.ratePlaceholder')"
            />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500">%</span>
          </div>
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.users.commissionRate.rateHint') }}</p>
        </div>

        <!-- Effective Rate Preview -->
        <div class="rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-900/30">
          <div class="flex items-center justify-between text-sm">
            <span class="text-blue-700 dark:text-blue-300">{{ t('admin.users.commissionRate.effectiveRate') }}:</span>
            <span class="font-bold text-blue-900 dark:text-blue-100">{{ formatPercent(effectiveRate) }}%</span>
          </div>
        </div>
      </template>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button
          type="submit"
          form="commission-rate-form"
          :disabled="submitting || loading || loadError"
          class="btn btn-primary"
        >
          {{ submitting ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
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
const loadError = ref(false) // Track load failure state
const useGlobalRate = ref(true)
const customRatePercent = ref(10) // Default 10%
const rateInfo = ref<UserCommissionRateInfo>({
  user_commission_rate: null,
  global_commission_rate: 0,
  effective_rate: 0
})

// Computed effective rate based on current selection
const effectiveRate = computed(() => {
  if (useGlobalRate.value) {
    return rateInfo.value.global_commission_rate
  }
  return customRatePercent.value / 100
})

// Format rate as percentage
const formatPercent = (rate: number) => {
  return (rate * 100).toFixed(2)
}

// Load commission rate info when modal opens
watch(() => props.show, async (show) => {
  if (show && props.user) {
    loading.value = true
    loadError.value = false
    try {
      const info = await adminAPI.users.getCommissionRate(props.user.id)
      rateInfo.value = info

      // Set initial form values
      if (info.user_commission_rate === null) {
        useGlobalRate.value = true
        customRatePercent.value = info.global_commission_rate * 100
      } else {
        useGlobalRate.value = false
        customRatePercent.value = info.user_commission_rate * 100
      }
    } catch (e: any) {
      console.error('Failed to load commission rate:', e)
      appStore.showError(e.response?.data?.detail || t('common.error'))
      loadError.value = true
    } finally {
      loading.value = false
    }
  }
}, { immediate: true })

// Handle form submission
const handleSubmit = async () => {
  if (!props.user || loadError.value) return

  // Validate custom rate (including NaN check)
  if (!useGlobalRate.value) {
    const rate = customRatePercent.value
    if (isNaN(rate) || rate < 0 || rate > 100) {
      appStore.showError(t('admin.users.commissionRate.invalidRate'))
      return
    }
  }

  submitting.value = true
  try {
    const rate = useGlobalRate.value ? null : customRatePercent.value / 100
    await adminAPI.users.updateCommissionRate(props.user.id, rate)
    appStore.showSuccess(t('common.success'))
    emit('success')
    emit('close')
  } catch (e: any) {
    console.error('Failed to update commission rate:', e)
    appStore.showError(e.response?.data?.detail || t('common.error'))
  } finally {
    submitting.value = false
  }
}
</script>
