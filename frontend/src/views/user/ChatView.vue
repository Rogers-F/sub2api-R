<template>
  <div class="flex h-screen overflow-hidden bg-paper-50 text-dust-800 dark:bg-ink-950 dark:text-pearl-100">
    <!-- Sidebar: fixed on desktop, drawer on mobile -->
    <div
      class="fixed inset-y-0 left-0 z-40 transform transition-transform duration-300 lg:static lg:z-auto lg:translate-x-0"
      :class="drawerOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <ConversationSidebar
        :conversations="chat.conversations"
        :current-id="chat.currentConversationId"
        :loading="chat.conversationsLoading"
        :has-subscription="hasUsableResource"
        :has-more="!!chat.nextConversationsCursor"
        :loading-more="chat.conversationsLoadingMore"
        @new="onNew"
        @select="onSelect"
        @delete="onDelete"
        @load-more="chat.loadMoreConversations"
      />
    </div>

    <!-- Mobile overlay -->
    <transition name="chat-fade">
      <div
        v-if="drawerOpen"
        class="fixed inset-0 z-30 bg-black/50 lg:hidden"
        @click="drawerOpen = false"
      />
    </transition>

    <!-- Main -->
    <div class="flex min-w-0 flex-1 flex-col">
      <ChatTopbar @toggle-sidebar="drawerOpen = !drawerOpen" />

      <NoSubscriptionGate v-if="showGate" />

      <template v-else>
        <ChatEmptyState v-if="showEmpty" @pick="onPickExample" />
        <ChatMessageList
          v-else
          :messages="chat.messages"
          :streaming="chat.isStreaming"
          :has-older="!!chat.messagesPrevCursor"
          :loading-older="chat.messagesLoadingMore"
          @load-older="chat.loadOlderMessages"
          @regenerate="chat.regenerate"
        />
        <ChatComposer
          ref="composerRef"
          :streaming="chat.isStreaming"
          :disabled="composerDisabled"
          @send="onSend"
          @stop="chat.stopGeneration"
        />
      </template>
    </div>

    <ConfirmDialog
      :show="!!pendingDeleteId"
      :title="t('chat.deleteConfirmTitle')"
      :message="t('chat.deleteConfirmBody')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="pendingDeleteId = null"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ConversationSidebar from '@/components/chat/ConversationSidebar.vue'
import ChatTopbar from '@/components/chat/ChatTopbar.vue'
import ChatMessageList from '@/components/chat/ChatMessageList.vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatEmptyState from '@/components/chat/ChatEmptyState.vue'
import NoSubscriptionGate from '@/components/chat/NoSubscriptionGate.vue'
import { useChatStore } from '@/stores/chat'
import { usePlaygroundStore } from '@/stores/playground'
import { useAppStore } from '@/stores/app'
import { usePlaygroundResources } from '@/composables/playground/usePlaygroundResources'

const { t } = useI18n()
const chat = useChatStore()
const playground = usePlaygroundStore()
const appStore = useAppStore()
const resources = usePlaygroundResources()

const drawerOpen = ref(false)
const composerRef = ref<InstanceType<typeof ChatComposer> | null>(null)
const pendingDeleteId = ref<number | null>(null)

// getAvailable() already filters groups by active subscription / permission, so a
// non-empty group list is the authoritative "this user can chat" signal.
const hasUsableResource = computed(() => playground.groups.length > 0)

// Show the gate only after resources have finished loading and none are usable.
const showGate = computed(() => !playground.resourcesLoading && !hasUsableResource.value)

const showEmpty = computed(() => chat.messages.length === 0 && !chat.messagesLoading)
const composerDisabled = computed(() => showGate.value || !playground.inputs.model)

function onNew() {
  chat.startNewConversation()
  drawerOpen.value = false
}

async function onSelect(id: number) {
  await chat.selectConversation(id)
  drawerOpen.value = false
}

function onDelete(id: number) {
  pendingDeleteId.value = id
}

async function confirmDelete() {
  const id = pendingDeleteId.value
  pendingDeleteId.value = null
  if (id != null) await chat.deleteConversation(id)
}

function onPickExample(prompt: string) {
  composerRef.value?.setText(prompt)
}

async function onSend(text: string) {
  await chat.sendMessage(text)
}

onMounted(async () => {
  await Promise.all([appStore.fetchPublicSettings(), resources.loadAll(), chat.loadConversations()])
  // Resolve models for the selected key (selectedKey derives from persisted apiKeyId).
  if (playground.selectedKey && playground.models.length === 0) {
    await resources.loadModelsForKey(playground.selectedKey.key)
  }
})
</script>

<style scoped>
.chat-fade-enter-active,
.chat-fade-leave-active {
  transition: opacity 0.2s ease;
}
.chat-fade-enter-from,
.chat-fade-leave-to {
  opacity: 0;
}
</style>
