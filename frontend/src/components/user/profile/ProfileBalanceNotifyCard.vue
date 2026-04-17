<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.balanceNotify.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.balanceNotify.description') }}
      </p>
    </div>

    <div class="space-y-6 px-6 py-6">
      <div class="flex items-center justify-between">
        <label class="input-label mb-0">{{ t('profile.balanceNotify.enabled') }}</label>
        <label class="relative inline-flex cursor-pointer items-center">
          <input
            v-model="notifyEnabled"
            type="checkbox"
            class="peer sr-only"
            @change="handleToggle"
          />
          <div
            class="h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-primary-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:after:translate-x-full dark:bg-gray-700 dark:peer-focus:ring-primary-800 dark:after:border-gray-600"
          ></div>
        </label>
      </div>

      <template v-if="notifyEnabled">
        <div>
          <label class="input-label">
            {{ t('profile.balanceNotify.threshold') }}
            <span class="ml-2 text-xs text-gray-400">
              {{ t('profile.balanceNotify.thresholdHint') }}
            </span>
          </label>
          <div class="flex items-center gap-2">
            <span class="text-gray-500">$</span>
            <input
              v-model.number="customThreshold"
              type="number"
              min="0"
              step="0.01"
              class="input flex-1"
              :placeholder="
                systemDefaultThreshold > 0
                  ? `${t('profile.balanceNotify.systemDefault')} $${systemDefaultThreshold}`
                  : t('profile.balanceNotify.thresholdPlaceholder')
              "
            />
            <button
              type="button"
              class="btn btn-primary btn-sm whitespace-nowrap"
              :disabled="savingThreshold"
              @click="handleThresholdUpdate"
            >
              {{ savingThreshold ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('profile.balanceNotify.extraEmails') }}</label>
          <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
            {{ t('profile.balanceNotify.extraEmailsHint') }}
          </p>

          <div v-if="normalizedEmailEntries.length > 0" class="mb-3 space-y-2">
            <div
              v-for="entry in normalizedEmailEntries"
              :key="entry.email"
              class="flex items-center justify-between rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700"
            >
              <div class="flex min-w-0 flex-1 items-center gap-2">
                <label class="relative inline-flex cursor-pointer items-center">
                  <input
                    type="checkbox"
                    class="peer sr-only"
                    :checked="!entry.disabled"
                    @change="handleEmailToggle(entry)"
                  />
                  <div
                    class="h-5 w-9 rounded-full bg-gray-200 peer-checked:bg-primary-600 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:after:translate-x-full dark:bg-gray-600 dark:after:border-gray-500"
                  ></div>
                </label>
                <span class="truncate text-sm text-gray-700 dark:text-gray-300">
                  {{ entry.email }}
                </span>
              </div>
              <div class="ml-3 flex items-center gap-2">
                <span class="text-xs" :class="entry.verified ? 'text-green-600 dark:text-green-400' : 'text-yellow-600 dark:text-yellow-400'">
                  {{
                    entry.verified
                      ? t('profile.balanceNotify.verified')
                      : t('profile.balanceNotify.unverified')
                  }}
                </span>
                <button
                  type="button"
                  class="text-xs text-red-500 hover:text-red-700"
                  @click="handleRemoveEmail(entry.email)"
                >
                  {{ t('profile.balanceNotify.removeEmail') }}
                </button>
              </div>
            </div>
          </div>

          <div
            v-if="pendingEmail"
            class="mb-3 rounded-lg border border-yellow-200 bg-yellow-50 px-3 py-3 dark:border-yellow-800 dark:bg-yellow-900/10"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm text-gray-700 dark:text-gray-300">
                  {{ pendingEmail }}
                </div>
                <div class="mt-1 text-xs text-yellow-700 dark:text-yellow-300">
                  {{ t('profile.balanceNotify.pendingHint') }}
                </div>
              </div>
              <div class="flex flex-1 items-center gap-2">
                <input
                  v-model="verificationCode"
                  type="text"
                  maxlength="6"
                  class="input w-24"
                  :placeholder="t('profile.balanceNotify.codePlaceholder')"
                />
                <button
                  type="button"
                  class="btn btn-primary btn-sm"
                  :disabled="verifyingEmail || verificationCode.trim().length !== 6"
                  @click="handleVerifyPendingEmail"
                >
                  {{ t('profile.balanceNotify.verify') }}
                </button>
                <button
                  v-if="countdown <= 0"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="sendingCode"
                  @click="handleResendCode"
                >
                  {{ t('profile.balanceNotify.resend') }}
                </button>
                <span v-else class="text-xs text-gray-400">{{ countdown }}s</span>
              </div>
            </div>
          </div>

          <div v-if="canAddMore" class="flex gap-2">
            <input
              v-model="newEmail"
              type="email"
              class="input flex-1"
              :placeholder="t('profile.balanceNotify.emailPlaceholder')"
              @keyup.enter="addNotifyEmail"
            />
            <button
              type="button"
              class="btn btn-secondary whitespace-nowrap"
              :disabled="sendingCode || !newEmail.trim()"
              @click="addNotifyEmail"
            >
              {{ t('common.add') }}
            </button>
          </div>
          <p v-else class="text-xs text-gray-400">
            {{ t('profile.balanceNotify.maxEmailsReached') }}
          </p>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { userAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import type { NotifyEmailEntry, User } from '@/types'

const maxNotifyEmails = 3
const verifyCountdownSeconds = 60

const props = defineProps<{
  enabled: boolean
  threshold: number | null
  extraEmails: NotifyEmailEntry[] | null
  systemDefaultThreshold: number
  userEmail: string
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const notifyEnabled = ref(props.enabled)
const customThreshold = ref<number | null>(props.threshold)
const emailEntries = ref<NotifyEmailEntry[]>(props.extraEmails ?? [])
const newEmail = ref('')
const pendingEmail = ref('')
const verificationCode = ref('')
const savingThreshold = ref(false)
const sendingCode = ref(false)
const verifyingEmail = ref(false)
const countdown = ref(0)

let countdownTimer: number | null = null

const normalizedEmailEntries = computed(() => emailEntries.value ?? [])
const canAddMore = computed(
  () => normalizedEmailEntries.value.length + (pendingEmail.value ? 1 : 0) < maxNotifyEmails
)

watch(
  () => props.enabled,
  (value) => {
    notifyEnabled.value = value
  }
)

watch(
  () => props.threshold,
  (value) => {
    customThreshold.value = value
  }
)

watch(
  () => props.extraEmails,
  (value) => {
    emailEntries.value = value ?? []
  }
)

onMounted(() => {
  if (!emailEntries.value.length && props.userEmail) {
    newEmail.value = props.userEmail
  }
})

onUnmounted(() => {
  stopCountdown()
})

function stopCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function startCountdown() {
  stopCountdown()
  countdown.value = verifyCountdownSeconds
  countdownTimer = window.setInterval(() => {
    if (countdown.value <= 1) {
      stopCountdown()
      countdown.value = 0
      return
    }
    countdown.value -= 1
  }, 1000)
}

function applyUpdatedUser(updatedUser: User) {
  authStore.user = updatedUser
  emailEntries.value = updatedUser.balance_notify_extra_emails ?? []
  notifyEnabled.value = updatedUser.balance_notify_enabled ?? notifyEnabled.value
  customThreshold.value = updatedUser.balance_notify_threshold ?? customThreshold.value
}

function normalizeEmail(value: string): string {
  return value.trim().toLowerCase()
}

function extractErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null) {
    const maybeMessage = (error as { message?: string }).message
    const maybeResponse = error as {
      response?: { data?: { detail?: string; message?: string } }
    }
    return (
      maybeResponse.response?.data?.detail ||
      maybeResponse.response?.data?.message ||
      maybeMessage ||
      fallback
    )
  }
  return fallback
}

function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
}

function emailExists(value: string): boolean {
  const target = normalizeEmail(value)
  return normalizedEmailEntries.value.some((entry) => normalizeEmail(entry.email) === target)
}

async function handleToggle() {
  try {
    const updatedUser = await userAPI.updateProfile({
      balance_notify_enabled: notifyEnabled.value
    })
    applyUpdatedUser(updatedUser)
  } catch (error: unknown) {
    notifyEnabled.value = !notifyEnabled.value
    appStore.showError(extractErrorMessage(error, t('profile.updateFailed')))
  }
}

async function handleThresholdUpdate() {
  savingThreshold.value = true
  try {
    const normalizedThreshold =
      customThreshold.value && customThreshold.value > 0 ? customThreshold.value : 0
    const updatedUser = await userAPI.updateProfile({
      balance_notify_threshold: normalizedThreshold
    })
    applyUpdatedUser(updatedUser)
    appStore.showSuccess(t('common.saved'))
  } catch (error: unknown) {
    appStore.showError(extractErrorMessage(error, t('profile.updateFailed')))
  } finally {
    savingThreshold.value = false
  }
}

async function requestVerifyCode(email: string) {
  sendingCode.value = true
  try {
    await userAPI.sendNotifyEmailCode(email)
    pendingEmail.value = email
    verificationCode.value = ''
    startCountdown()
    appStore.showSuccess(t('profile.balanceNotify.codeSent'))
  } catch (error: unknown) {
    appStore.showError(extractErrorMessage(error, t('profile.balanceNotify.sendCodeFailed')))
  } finally {
    sendingCode.value = false
  }
}

async function addNotifyEmail() {
  const email = newEmail.value.trim()
  if (!isValidEmail(email)) {
    appStore.showError(t('profile.balanceNotify.invalidEmail'))
    return
  }
  if (emailExists(email)) {
    appStore.showError(t('profile.balanceNotify.duplicateEmail'))
    return
  }
  if (!canAddMore.value) {
    appStore.showError(t('profile.balanceNotify.maxEmailsReached'))
    return
  }
  await requestVerifyCode(email)
}

async function handleVerifyPendingEmail() {
  if (!pendingEmail.value || verificationCode.value.trim().length !== 6) {
    return
  }
  verifyingEmail.value = true
  try {
    const updatedUser = await userAPI.verifyNotifyEmail(
      pendingEmail.value,
      verificationCode.value.trim()
    )
    applyUpdatedUser(updatedUser)
    pendingEmail.value = ''
    verificationCode.value = ''
    newEmail.value = ''
    stopCountdown()
    countdown.value = 0
    appStore.showSuccess(t('profile.balanceNotify.verifySuccess'))
  } catch (error: unknown) {
    appStore.showError(extractErrorMessage(error, t('profile.balanceNotify.verifyFailed')))
  } finally {
    verifyingEmail.value = false
  }
}

async function handleResendCode() {
  if (!pendingEmail.value) {
    return
  }
  await requestVerifyCode(pendingEmail.value)
}

async function handleEmailToggle(entry: NotifyEmailEntry) {
  try {
    const updatedUser = await userAPI.toggleNotifyEmail(entry.email, !entry.disabled)
    applyUpdatedUser(updatedUser)
  } catch (error: unknown) {
    appStore.showError(extractErrorMessage(error, t('common.error')))
  }
}

async function handleRemoveEmail(email: string) {
  try {
    const updatedUser = await userAPI.removeNotifyEmail(email)
    applyUpdatedUser(updatedUser)
    appStore.showSuccess(t('profile.balanceNotify.removeSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractErrorMessage(error, t('common.error')))
  }
}
</script>
