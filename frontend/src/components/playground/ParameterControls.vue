<template>
  <div class="space-y-4">
    <h2 class="text-sm font-semibold text-primary-fg">{{ t('playground.parameters.title') }}</h2>

    <ParameterRow
      :label="t('playground.parameters.temperature')"
      :model-value="store.inputs.temperature"
      :enabled="store.parameterEnabled.temperature"
      :min="0"
      :max="2"
      :step="0.05"
      @update:model-value="(v) => store.setInput('temperature', v)"
      @update:enabled="(v) => store.setParameterEnabled('temperature', v)"
    />
    <ParameterRow
      :label="t('playground.parameters.topP')"
      :model-value="store.inputs.top_p"
      :enabled="store.parameterEnabled.top_p"
      :min="0"
      :max="1"
      :step="0.05"
      @update:model-value="(v) => store.setInput('top_p', v)"
      @update:enabled="(v) => store.setParameterEnabled('top_p', v)"
    />
    <ParameterRow
      :label="t('playground.parameters.maxTokens')"
      :model-value="store.inputs.max_tokens"
      :enabled="store.parameterEnabled.max_tokens"
      :min="1"
      :max="32768"
      :step="1"
      :integer="true"
      @update:model-value="(v) => store.setInput('max_tokens', v)"
      @update:enabled="(v) => store.setParameterEnabled('max_tokens', v)"
    />
    <ParameterRow
      :label="t('playground.parameters.frequencyPenalty')"
      :model-value="store.inputs.frequency_penalty"
      :enabled="store.parameterEnabled.frequency_penalty"
      :min="-2"
      :max="2"
      :step="0.05"
      @update:model-value="(v) => store.setInput('frequency_penalty', v)"
      @update:enabled="(v) => store.setParameterEnabled('frequency_penalty', v)"
    />
    <ParameterRow
      :label="t('playground.parameters.presencePenalty')"
      :model-value="store.inputs.presence_penalty"
      :enabled="store.parameterEnabled.presence_penalty"
      :min="-2"
      :max="2"
      :step="0.05"
      @update:model-value="(v) => store.setInput('presence_penalty', v)"
      @update:enabled="(v) => store.setParameterEnabled('presence_penalty', v)"
    />
    <ParameterRow
      :label="t('playground.parameters.seed')"
      :model-value="store.inputs.seed ?? 0"
      :enabled="store.parameterEnabled.seed"
      :min="0"
      :max="999999999"
      :step="1"
      :integer="true"
      @update:model-value="(v) => store.setInput('seed', v)"
      @update:enabled="(v) => store.setParameterEnabled('seed', v)"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { usePlaygroundStore } from '@/stores/playground'
import ParameterRow from './ParameterRow.vue'

const store = usePlaygroundStore()
const { t } = useI18n()
</script>
