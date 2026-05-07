<template>
  <BaseDialog :show="show" :title="title" width="narrow" @close="handleCancel">
    <div class="space-y-4">
      <p class="text-sm text-dust-600 dark:text-pearl-300">{{ message }}</p>
      <slot></slot>
    </div>

    <template #footer>
      <div class="flex justify-end space-x-3">
        <button
          @click="handleCancel"
          type="button"
          class="rounded-md border border-paper-300 bg-white px-4 py-2 text-sm font-medium text-dust-700 hover:bg-paper-50 focus:outline-none focus:ring-2 focus:ring-gold-500 focus:ring-offset-2 dark:border-ink-600 dark:bg-ink-700 dark:text-pearl-100 dark:hover:bg-ink-600 dark:focus:ring-gold-300 dark:focus:ring-offset-ink-800"
        >
          {{ cancelText }}
        </button>
        <button
          @click="handleConfirm"
          type="button"
          :class="[
            'rounded-md px-4 py-2 text-sm font-medium text-white focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-ink-800',
            danger
              ? 'bg-coral-500 hover:bg-coral-600 focus:ring-coral-500'
              : 'bg-gold-500 hover:bg-gold-600 focus:ring-gold-500'
          ]"
        >
          {{ confirmText }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from './BaseDialog.vue'

const { t } = useI18n()

interface Props {
  show: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

interface Emits {
  (e: 'confirm'): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  danger: false
})

const confirmText = computed(() => props.confirmText || t('common.confirm'))
const cancelText = computed(() => props.cancelText || t('common.cancel'))

const emit = defineEmits<Emits>()

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
}
</script>
