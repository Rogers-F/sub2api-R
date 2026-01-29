<template>
  <div v-if="contactInfo" class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.contactUs') }}</h2>
    </div>
    <div class="p-4">
      <div class="space-y-3 text-sm">
        <!-- Parse contact info lines -->
        <div
          v-for="(line, index) in contactLines"
          :key="index"
          class="flex items-center gap-3"
        >
          <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 dark:bg-dark-700">
            <Icon :name="getContactIcon(line)" size="sm" class="text-gray-500 dark:text-gray-400" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="text-gray-700 dark:text-gray-300">{{ line }}</p>
          </div>
          <button
            @click="copyText(line)"
            class="flex h-7 w-7 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-300"
            :title="t('common.copy')"
          >
            <Icon :name="copiedIndex === index ? 'check' : 'copy'" size="xs" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const contactInfo = computed(() => appStore.cachedPublicSettings?.contact_info || appStore.contactInfo || '')

const contactLines = computed(() => {
  if (!contactInfo.value) return []
  return contactInfo.value.split('\n').filter(line => line.trim())
})

const copiedIndex = ref<number | null>(null)

const getContactIcon = (line: string): string => {
  const lowerLine = line.toLowerCase()
  if (lowerLine.includes('qq') || lowerLine.includes('群')) return 'users'
  if (lowerLine.includes('微信') || lowerLine.includes('wechat')) return 'chat'
  if (lowerLine.includes('邮箱') || lowerLine.includes('email') || lowerLine.includes('@')) return 'mail'
  if (lowerLine.includes('电话') || lowerLine.includes('phone') || lowerLine.includes('tel')) return 'phone'
  if (lowerLine.includes('telegram') || lowerLine.includes('tg')) return 'message'
  if (lowerLine.includes('discord')) return 'users'
  return 'info'
}

const copyText = async (text: string) => {
  const index = contactLines.value.indexOf(text)
  try {
    await navigator.clipboard.writeText(text)
    copiedIndex.value = index
    setTimeout(() => {
      copiedIndex.value = null
    }, 2000)
  } catch (e) {
    console.error('Failed to copy:', e)
  }
}
</script>
