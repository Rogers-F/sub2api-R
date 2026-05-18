<template>
  <div class="border-t hairline px-4 py-3">
    <div class="flex items-end gap-2">
      <TextArea
        v-model="draft"
        :placeholder="t('playground.chat.placeholderInput')"
        :rows="2"
        :disabled="isSending"
        class="flex-1"
        @keydown="onKeydown"
      />
      <div class="flex flex-col gap-2">
        <label class="flex items-center gap-2 text-xs text-secondary-fg">
          <Toggle v-model="streamProxy" />
          <span>{{ t('playground.chat.streamToggle') }}</span>
        </label>
        <div class="flex gap-2">
          <button
            v-if="isSending"
            class="btn btn-danger btn-sm"
            @click="onStop"
          >
            <Icon name="x" size="sm" class="mr-1" />
            {{ t('playground.chat.stop') }}
          </button>
          <button
            v-else
            class="btn btn-gold btn-sm"
            :disabled="!canSend"
            @click="onSend"
          >
            <Icon name="arrowRight" size="sm" class="mr-1" />
            {{ t('playground.chat.send') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import TextArea from '@/components/common/TextArea.vue'
import Toggle from '@/components/common/Toggle.vue'
import { STREAM_STATUS } from '@/constants/playground'
import { usePlaygroundStore } from '@/stores/playground'
import { useChatStream } from '@/composables/playground/useChatStream'

const store = usePlaygroundStore()
const chat = useChatStream()
const { t } = useI18n()

const draft = ref<string>('')
const isSending = computed(() => store.status === STREAM_STATUS.STREAMING)

const streamProxy = computed({
  get: () => store.inputs.stream,
  set: (v: boolean) => store.setInput('stream', v)
})

const canSend = computed(
  () => draft.value.trim().length > 0 && !!store.apiKey && !!store.inputs.model
)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    onSend()
  }
}

async function onSend() {
  if (!canSend.value) return
  const text = draft.value.trim()
  draft.value = ''
  await chat.sendMessage(text)
}

function onStop() {
  chat.stopGeneration()
}
</script>
