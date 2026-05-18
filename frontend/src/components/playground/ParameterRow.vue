<template>
  <div :class="['rounded-lg p-2 hairline', enabled ? '' : 'opacity-60']">
    <div class="mb-1 flex items-center justify-between gap-2">
      <label class="text-xs font-medium text-primary-fg">{{ label }}</label>
      <div class="flex items-center gap-2">
        <input
          type="number"
          :value="localValue"
          :min="min"
          :max="max"
          :step="step"
          :disabled="!enabled"
          class="w-20 rounded-md bg-paper-50 px-2 py-1 text-right text-xs text-primary-fg hairline disabled:opacity-50 dark:bg-ink-900"
          @input="onNumberInput"
        />
        <Toggle :model-value="enabled" @update:model-value="emit('update:enabled', $event)" />
      </div>
    </div>
    <input
      type="range"
      :value="localValue"
      :min="min"
      :max="max"
      :step="step"
      :disabled="!enabled"
      class="w-full accent-gold-500 disabled:opacity-40"
      @input="onRangeInput"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import Toggle from '@/components/common/Toggle.vue'

const props = defineProps<{
  label: string
  modelValue: number
  enabled: boolean
  min: number
  max: number
  step: number
  integer?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: number): void
  (e: 'update:enabled', value: boolean): void
}>()

const localValue = computed(() => props.modelValue)

function coerce(raw: string): number | null {
  const n = props.integer ? parseInt(raw, 10) : parseFloat(raw)
  if (Number.isNaN(n)) return null
  if (n < props.min) return props.min
  if (n > props.max) return props.max
  return n
}

function onRangeInput(e: Event) {
  const v = coerce((e.target as HTMLInputElement).value)
  if (v !== null) emit('update:modelValue', v)
}

function onNumberInput(e: Event) {
  const v = coerce((e.target as HTMLInputElement).value)
  if (v !== null) emit('update:modelValue', v)
}
</script>
