<template>
  <AppLayout>
    <div class="flex h-[calc(100vh-7rem)] flex-col lg:flex-row gap-4">
      <!-- 左侧设置面板 -->
      <aside
        class="w-full lg:w-80 flex-shrink-0 panel rounded-2xl p-4 overflow-y-auto"
      >
        <div class="mb-4">
          <h1 class="page-title text-xl">{{ t('playground.title') }}</h1>
          <p class="page-description text-xs">{{ t('playground.description') }}</p>
        </div>

        <KeyGroupModelPicker />

        <div class="mt-6">
          <ParameterControls />
        </div>

        <div class="mt-6 flex flex-col gap-2">
          <button
            class="btn btn-secondary btn-sm"
            :disabled="store.messages.length === 0"
            @click="showClearConfirm = true"
          >
            <Icon name="trash" size="sm" class="mr-1" />
            {{ t('playground.chat.clearAll') }}
          </button>
        </div>
      </aside>

      <!-- 主对话区 -->
      <section class="panel flex-1 flex flex-col rounded-2xl overflow-hidden">
        <MessageList />
        <PromptInput />
      </section>
    </div>

    <ConfirmDialog
      :show="showClearConfirm"
      :title="t('playground.chat.clearConfirmTitle')"
      :message="t('playground.chat.clearConfirmBody')"
      :danger="true"
      @confirm="onConfirmClear"
      @cancel="showClearConfirm = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import KeyGroupModelPicker from '@/components/playground/KeyGroupModelPicker.vue'
import MessageList from '@/components/playground/MessageList.vue'
import PromptInput from '@/components/playground/PromptInput.vue'
import ParameterControls from '@/components/playground/ParameterControls.vue'
import { usePlaygroundStore } from '@/stores/playground'
import { usePlaygroundResources } from '@/composables/playground/usePlaygroundResources'

const store = usePlaygroundStore()
const resources = usePlaygroundResources()
const { t } = useI18n()

const showClearConfirm = ref(false)

onMounted(async () => {
  await resources.loadAll()
  // 资源就绪后用 selectedKey 拉模型（apiKey 是从 selectedKey 派生的 getter）
  if (store.selectedKey && store.models.length === 0) {
    await resources.loadModelsForKey(store.selectedKey.key)
  }
})

function onConfirmClear() {
  store.clearMessages()
  showClearConfirm.value = false
}
</script>
