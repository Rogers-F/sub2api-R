<template>
  <div class="flex flex-1 items-center justify-center px-6 py-6">
    <div
      class="w-full max-w-[440px] rounded-[20px] border p-9 text-center hairline bg-paper-0 dark:bg-ink-900"
    >
      <div
        class="mx-auto mb-[18px] flex h-14 w-14 items-center justify-center rounded-full border border-gold-400 bg-gold-500/[0.12] text-gold-700 dark:border-gold-600 dark:text-gold-300"
      >
        <Icon name="lock" size="lg" />
      </div>
      <h2 class="mb-2 text-[19px] font-bold text-dust-900 dark:text-pearl-50">
        {{ t('chat.noSub.title') }}
      </h2>
      <p class="mb-6 text-[13.5px] leading-relaxed text-dust-400 dark:text-pearl-400">
        {{ t('chat.noSub.subtitle') }}
      </p>

      <button
        v-if="purchaseEnabled"
        type="button"
        class="mb-3 block w-full rounded-xl bg-gradient-to-br from-gold-400 to-gold-600 px-4 py-3 text-sm font-bold text-dust-900 transition-opacity cursor-pointer hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60"
        @click="goPurchase"
      >
        {{ t('chat.noSub.buy') }}
      </button>

      <div class="flex gap-3">
        <button
          type="button"
          class="flex-1 rounded-xl border px-3 py-2.5 text-[13px] font-semibold transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 border-paper-300 bg-paper-50 text-dust-700 hover:border-gold-400 dark:border-ink-600 dark:bg-ink-800 dark:text-pearl-200 dark:hover:border-gold-600"
          @click="goRedeem"
        >
          {{ t('chat.noSub.redeem') }}
        </button>
        <a
          v-if="contactInfo"
          :href="contactHref"
          target="_blank"
          rel="noopener noreferrer"
          class="flex-1 rounded-xl border px-3 py-2.5 text-center text-[13px] font-semibold transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 border-paper-300 bg-paper-50 text-dust-700 hover:border-gold-400 dark:border-ink-600 dark:bg-ink-800 dark:text-pearl-200 dark:hover:border-gold-600"
        >
          {{ t('chat.noSub.contact') }}
        </a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()

const purchaseEnabled = computed(
  () => !!appStore.cachedPublicSettings?.purchase_subscription_enabled
)
const purchaseUrl = computed(() => appStore.cachedPublicSettings?.purchase_subscription_url || '')
const contactInfo = computed(() => appStore.contactInfo)

// Contact info may be a URL, an email, or a plain handle. Build a sensible href.
const contactHref = computed(() => {
  const value = contactInfo.value
  if (!value) return '#'
  if (/^https?:\/\//i.test(value)) return value
  if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) return `mailto:${value}`
  return value
})

function goPurchase() {
  // Prefer an external purchase URL if configured; otherwise use the in-app page.
  if (purchaseUrl.value) {
    window.open(purchaseUrl.value, '_blank', 'noopener,noreferrer')
    return
  }
  void router.push('/purchase')
}

function goRedeem() {
  void router.push('/redeem')
}
</script>
