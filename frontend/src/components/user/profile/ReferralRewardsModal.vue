<template>
  <BaseDialog
    :show="show"
    :title="t('profile.referral.rewardsTitle')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
      </div>

      <!-- Empty state -->
      <div v-else-if="rewards.length === 0" class="text-center py-8">
        <Icon name="gift" size="xl" class="mx-auto text-gray-300 dark:text-dark-600" />
        <p class="mt-2 text-gray-500 dark:text-dark-400">{{ t('profile.referral.noRewards') }}</p>
      </div>

      <!-- Rewards list -->
      <div v-else class="max-h-96 overflow-y-auto">
        <table class="w-full">
          <thead class="sticky top-0 bg-white dark:bg-dark-800">
            <tr class="border-b border-gray-200 dark:border-dark-700">
              <th class="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('profile.referral.rewardType.register') }}
              </th>
              <th class="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('profile.referral.referee') }}
              </th>
              <th class="px-4 py-3 text-right text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('profile.referral.amount') }}
              </th>
              <th class="px-4 py-3 text-right text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('profile.referral.time') }}
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
                <span
                  :class="[
                    'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                    reward.reward_type === 'register'
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                      : 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                  ]"
                >
                  {{ reward.reward_type === 'register' ? t('profile.referral.rewardType.register') : t('profile.referral.rewardType.commission') }}
                </span>
              </td>
              <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">
                {{ reward.referee_email || '-' }}
              </td>
              <td class="px-4 py-3 text-right text-sm font-medium text-green-600 dark:text-green-400">
                +${{ reward.reward_amount.toFixed(2) }}
              </td>
              <td class="px-4 py-3 text-right text-sm text-gray-500 dark:text-dark-400">
                {{ formatDate(reward.created_at) }}
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Pagination -->
        <div v-if="totalPages > 1" class="mt-4 flex justify-center">
          <nav class="flex items-center gap-1">
            <button
              type="button"
              class="px-3 py-1 rounded text-sm"
              :disabled="page === 1"
              :class="page === 1 ? 'text-gray-300 dark:text-dark-600' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700'"
              @click="loadPage(page - 1)"
            >
              {{ t('common.previous') }}
            </button>
            <span class="px-3 py-1 text-sm text-gray-600 dark:text-dark-300">
              {{ page }} / {{ totalPages }}
            </span>
            <button
              type="button"
              class="px-3 py-1 rounded text-sm"
              :disabled="page === totalPages"
              :class="page === totalPages ? 'text-gray-300 dark:text-dark-600' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700'"
              @click="loadPage(page + 1)"
            >
              {{ t('common.next') }}
            </button>
          </nav>
        </div>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-outline" @click="emit('close')">
        {{ t('profile.referral.close') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { referralAPI } from '@/api/referral'
import type { ReferralReward } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()

const loading = ref(true)
const rewards = ref<ReferralReward[]>([])
const page = ref(1)
const pageSize = 10
const totalPages = ref(1)

const loadPage = async (newPage: number) => {
  if (newPage < 1 || newPage > totalPages.value) return
  page.value = newPage
  loading.value = true
  try {
    const res = await referralAPI.getReferralRewards(page.value, pageSize)
    rewards.value = res.items
    totalPages.value = Math.ceil(res.total / pageSize)
  } catch (error) {
    console.error('Failed to load rewards:', error)
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr)
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(() => {
  loadPage(1)
})
</script>
