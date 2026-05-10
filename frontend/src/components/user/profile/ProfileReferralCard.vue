<template>
  <div class="card">
    <div class="border-b hairline px-6 py-4">
      <h2 class="text-lg font-medium text-dust-900 dark:text-white">
        {{ t('profile.referral.title') }}
      </h2>
      <p class="mt-1 text-sm text-secondary-fg">
        {{ t('profile.referral.description') }}
      </p>
    </div>
    <div class="px-6 py-6">
      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-gold-500"></div>
      </div>

      <!-- System disabled -->
      <div v-else-if="!settings?.enabled" class="flex items-center gap-4 py-4">
        <div class="flex-shrink-0 rounded-full bg-paper-100 p-3 dark:bg-ink-700">
          <Icon name="userPlus" size="lg" class="text-dust-400" />
        </div>
        <div>
          <p class="font-medium text-dust-700 dark:text-pearl-100">
            {{ t('profile.referral.systemDisabled') }}
          </p>
        </div>
      </div>

      <!-- Referral enabled -->
      <div v-else class="space-y-6">
        <!-- My referral code -->
        <div>
          <label class="block text-sm font-medium text-dust-700 dark:text-pearl-200 mb-2">
            {{ t('profile.referral.myCode') }}
          </label>
          <div class="flex items-center gap-2">
            <div class="flex-1 rounded-lg border hairline bg-paper-50 px-4 py-3 font-mono text-lg dark:bg-ink-800">
              {{ referralInfo?.referral_code || '-' }}
            </div>
            <button
              type="button"
              class="btn btn-outline"
              :title="t('profile.referral.copyCode')"
              @click="copyCode"
            >
              <Icon name="copy" size="md" />
            </button>
            <button
              type="button"
              class="btn btn-outline"
              :title="t('profile.referral.copyLink')"
              @click="copyLink"
            >
              <Icon name="link" size="md" />
            </button>
          </div>
        </div>

        <!-- Stats -->
        <div class="grid grid-cols-2 gap-4">
          <div class="rounded-lg border hairline bg-paper-50 p-4 dark:bg-ink-800">
            <p class="text-sm text-secondary-fg">{{ t('profile.referral.invitedCount') }}</p>
            <p class="mt-1 text-2xl font-semibold text-dust-900 dark:text-white">
              {{ referralInfo?.total_invited || 0 }}
            </p>
          </div>
          <div class="rounded-lg border hairline bg-paper-50 p-4 dark:bg-ink-800">
            <p class="text-sm text-secondary-fg">{{ t('profile.referral.totalRewards') }}</p>
            <p class="mt-1 text-2xl font-semibold text-green-600 dark:text-green-400">
              ${{ (referralInfo?.total_reward || 0).toFixed(2) }}
            </p>
          </div>
        </div>

        <!-- View rewards history -->
        <button
          type="button"
          class="btn btn-outline w-full"
          @click="showRewardsModal = true"
        >
          <Icon name="list" size="md" class="mr-2" />
          {{ t('profile.referral.viewRewards') }}
        </button>
      </div>
    </div>

    <!-- Rewards Modal -->
    <ReferralRewardsModal
      v-if="showRewardsModal"
      :show="showRewardsModal"
      @close="showRewardsModal = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { referralAPI } from '@/api/referral'
import { copyToClipboard } from '@/utils/clipboard'
import Icon from '@/components/icons/Icon.vue'
import ReferralRewardsModal from './ReferralRewardsModal.vue'
import type { ReferralInfo, ReferralSettings } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const referralInfo = ref<ReferralInfo | null>(null)
const settings = ref<ReferralSettings | null>(null)
const showRewardsModal = ref(false)

async function loadData(): Promise<void> {
  loading.value = true
  try {
    const [infoRes, settingsRes] = await Promise.all([
      referralAPI.getReferralInfo(),
      referralAPI.getReferralSettings()
    ])
    referralInfo.value = infoRes
    settings.value = settingsRes
  } catch (error) {
    console.error('Failed to load referral data:', error)
  } finally {
    loading.value = false
  }
}

async function copyCode(): Promise<void> {
  if (!referralInfo.value?.referral_code) return
  const success = await copyToClipboard(referralInfo.value.referral_code)
  if (success) {
    appStore.showSuccess(t('profile.referral.codeCopied'))
  } else {
    appStore.showError(t('common.copyFailed'))
  }
}

async function copyLink(): Promise<void> {
  if (!referralInfo.value?.referral_code) return
  const link = `${window.location.origin}/register?ref=${referralInfo.value.referral_code}`
  const success = await copyToClipboard(link)
  if (success) {
    appStore.showSuccess(t('profile.referral.linkCopied'))
  } else {
    appStore.showError(t('common.copyFailed'))
  }
}

onMounted(loadData)
</script>
