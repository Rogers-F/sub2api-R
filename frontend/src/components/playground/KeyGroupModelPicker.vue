<template>
  <div class="space-y-4">
    <div>
      <label class="input-label">{{ t('playground.pickers.group') }}</label>
      <Select
        :model-value="store.inputs.group"
        :options="groupOptions"
        :placeholder="t('playground.pickers.selectGroup')"
        :disabled="store.resourcesLoading || modelsLoading || creatingKey"
        @update:model-value="onGroupChange"
      />
    </div>

    <div>
      <label class="input-label">{{ t('playground.pickers.apiKey') }}</label>
      <div class="flex flex-col gap-2">
        <Select
          :model-value="store.inputs.apiKeyId"
          :options="keyOptions"
          :placeholder="
            keyOptions.length === 0
              ? t('playground.pickers.noKeysForGroup')
              : t('playground.pickers.selectKey')
          "
          :disabled="!store.currentGroup || creatingKey"
          @update:model-value="onKeyChange"
        />
        <button
          v-if="store.currentGroup && keyOptions.length === 0"
          class="btn btn-secondary btn-sm"
          :disabled="creatingKey"
          @click="onCreateKey"
        >
          <Icon name="plus" size="sm" class="mr-1" />
          {{ creatingKey ? t('playground.pickers.creatingKey') : t('playground.pickers.createPlaygroundKey') }}
        </button>
      </div>
    </div>

    <div>
      <div class="mb-1 flex items-center justify-between">
        <label class="input-label">{{ t('playground.pickers.model') }}</label>
        <button
          v-if="store.apiKey"
          class="text-xs text-secondary-fg hover:text-primary-fg transition-colors"
          :title="t('playground.pickers.refreshModels')"
          @click="onRefreshModels"
        >
          <Icon name="refresh" size="sm" :class="modelsLoading ? 'animate-spin' : ''" />
        </button>
      </div>
      <Select
        :model-value="store.inputs.model"
        :options="store.models"
        :placeholder="
          !store.apiKey
            ? t('playground.pickers.modelEmpty')
            : modelsLoading
              ? t('playground.pickers.modelLoading')
              : t('playground.pickers.selectModel')
        "
        :disabled="!store.apiKey || modelsLoading"
        :searchable="true"
        @update:model-value="(v) => store.setInput('model', String(v ?? ''))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import { usePlaygroundStore } from '@/stores/playground'
import { usePlaygroundResources } from '@/composables/playground/usePlaygroundResources'

const store = usePlaygroundStore()
const resources = usePlaygroundResources()
const { t } = useI18n()

const creatingKey = ref(false)
const modelsLoading = ref(false)
// 用单调递增的 token 守护快速切换 group/key 时的请求竞态
let requestSeq = 0

const groupOptions = computed(() =>
  store.groups.map((g) => ({ value: g.name, label: g.name }))
)

const keyOptions = computed(() =>
  store.apiKeysForCurrentGroup.map((k) => ({
    value: k.id,
    label: `${k.name} · ${maskKey(k.key)}`
  }))
)

function maskKey(k: string) {
  if (!k) return ''
  if (k.length <= 12) return k
  return `${k.slice(0, 7)}…${k.slice(-4)}`
}

async function onGroupChange(value: string | number | boolean | null) {
  const myToken = ++requestSeq
  const groupName = String(value ?? '')
  store.setInput('group', groupName)
  store.setInput('apiKeyId', null)
  store.setInput('model', '')
  store.models = []

  const group = store.groups.find((g) => g.name === groupName)
  if (!group) return

  let key = resources.findActiveKeyForGroup(group.id)
  if (!key) {
    creatingKey.value = true
    try {
      key = await resources.createPlaygroundKey(group.id)
    } finally {
      creatingKey.value = false
    }
  }
  if (key && myToken === requestSeq) {
    await selectKey(key.id, myToken)
  }
}

function onKeyChange(value: string | number | boolean | null) {
  const id = typeof value === 'number' ? value : Number(value)
  if (Number.isNaN(id)) {
    store.setInput('apiKeyId', null)
    return
  }
  void selectKey(id, ++requestSeq)
}

async function selectKey(id: number, token: number) {
  store.setInput('apiKeyId', id)
  const key = store.apiKeys.find((k) => k.id === id)
  if (!key) return
  modelsLoading.value = true
  try {
    await resources.loadModelsForKey(key.key)
  } finally {
    modelsLoading.value = false
  }
  // 加载结束时若已被新的请求取代，则丢弃本次结果
  if (token !== requestSeq) return
  if (!store.inputs.model && store.models.length > 0) {
    store.setInput('model', store.models[0].value)
  }
}

async function onCreateKey() {
  if (!store.currentGroup) return
  creatingKey.value = true
  try {
    const key = await resources.createPlaygroundKey(store.currentGroup.id)
    if (key) await selectKey(key.id, ++requestSeq)
  } finally {
    creatingKey.value = false
  }
}

async function onRefreshModels() {
  if (!store.apiKey) return
  modelsLoading.value = true
  try {
    await resources.loadModelsForKey(store.apiKey)
  } finally {
    modelsLoading.value = false
  }
}
</script>
