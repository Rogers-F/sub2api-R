<template>
  <aside
    class="flex h-full w-72 flex-shrink-0 flex-col border-r hairline bg-paper-100 dark:bg-ink-900"
  >
    <!-- Top: brand + new + search -->
    <div class="p-4">
      <div class="mb-4 flex items-center gap-2.5">
        <div
          class="flex h-[30px] w-[30px] items-center justify-center rounded-[9px] bg-gradient-to-br from-gold-400 to-gold-600 text-[15px] font-extrabold text-dust-900"
        >
          {{ brandInitial }}
        </div>
        <span class="text-[15px] font-bold tracking-wide text-dust-900 dark:text-pearl-50">
          {{ t('chat.brandTitle') }}
        </span>
      </div>

      <button
        type="button"
        class="flex w-full items-center justify-center gap-2 rounded-xl border border-gold-400 bg-gold-500/[0.08] px-3 py-2.5 text-[13.5px] font-semibold text-gold-700 transition-colors cursor-pointer hover:bg-gold-500/[0.16] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 dark:border-gold-600 dark:text-gold-200"
        @click="emit('new')"
      >
        <Icon name="plus" size="sm" />
        {{ t('chat.new') }}
      </button>

      <label class="sr-only" for="chat-search">{{ t('chat.search') }}</label>
      <div
        class="mt-3 flex items-center gap-2 rounded-[10px] border hairline bg-paper-0 px-3 py-2 dark:bg-ink-800"
      >
        <Icon name="search" size="sm" class="text-dust-400 dark:text-pearl-400" />
        <input
          id="chat-search"
          v-model="query"
          type="text"
          :placeholder="t('chat.search')"
          class="w-full bg-transparent text-[13px] text-dust-700 placeholder:text-dust-400 focus:outline-none dark:text-pearl-100 dark:placeholder:text-pearl-500"
        />
      </div>
    </div>

    <!-- Conversation list, grouped -->
    <div class="flex-1 overflow-y-auto px-2.5 pb-2">
      <template v-if="todayItems.length">
        <div
          class="px-1.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-dust-400 dark:text-pearl-500"
        >
          {{ t('chat.todayGroup') }}
        </div>
        <ConversationItem
          v-for="conv in todayItems"
          :key="conv.id"
          :conversation="conv"
          :active="conv.id === currentId"
          @select="emit('select', $event)"
          @delete="emit('delete', $event)"
        />
      </template>

      <template v-if="earlierItems.length">
        <div
          class="mt-2 px-1.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-dust-400 dark:text-pearl-500"
        >
          {{ t('chat.earlierGroup') }}
        </div>
        <ConversationItem
          v-for="conv in earlierItems"
          :key="conv.id"
          :conversation="conv"
          :active="conv.id === currentId"
          @select="emit('select', $event)"
          @delete="emit('delete', $event)"
        />
      </template>

      <!-- Load more (cursor pagination); hidden while filtering a local search. -->
      <button
        v-if="hasMore && !query"
        type="button"
        :disabled="loadingMore"
        class="mt-2 flex w-full items-center justify-center rounded-lg px-3 py-2 text-[12.5px] font-medium text-dust-500 transition-colors cursor-pointer hover:bg-paper-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-500/60 disabled:cursor-not-allowed disabled:opacity-60 dark:text-pearl-400 dark:hover:bg-ink-800"
        @click="emit('load-more')"
      >
        {{ loadingMore ? t('common.loading') : t('chat.loadMore') }}
      </button>

      <p
        v-if="!loading && filtered.length === 0"
        class="px-2 py-6 text-center text-xs text-dust-400 dark:text-pearl-500"
      >
        {{ query ? t('chat.noResults') : t('chat.emptyList') }}
      </p>
    </div>

    <!-- User footer -->
    <div class="flex items-center gap-2.5 border-t hairline px-4 py-3">
      <div
        class="flex h-[30px] w-[30px] flex-shrink-0 items-center justify-center rounded-full bg-aurora-600 text-xs font-bold text-white"
      >
        {{ userInitial }}
      </div>
      <div class="min-w-0">
        <div class="truncate text-[13px] font-semibold text-dust-800 dark:text-pearl-100">
          {{ displayName }}
        </div>
        <div
          :class="[
            'truncate text-[11px]',
            hasSubscription
              ? 'text-mint-600 dark:text-mint-400'
              : 'text-coral-600 dark:text-coral-400'
          ]"
        >
          {{ subscriptionLabel }}
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import ConversationItem from './ConversationItem.vue'
import { useAuthStore } from '@/stores/auth'
import type { Conversation } from '@/types/chat'

const props = defineProps<{
  conversations: Conversation[]
  currentId: number | null
  loading: boolean
  hasSubscription: boolean
  hasMore?: boolean
  loadingMore?: boolean
}>()
const emit = defineEmits<{
  (e: 'new'): void
  (e: 'select', id: number): void
  (e: 'delete', id: number): void
  (e: 'load-more'): void
}>()

const { t } = useI18n()
const authStore = useAuthStore()

const query = ref('')

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.conversations
  return props.conversations.filter((c) => (c.title || '').toLowerCase().includes(q))
})

// Partition into today / earlier in a single pass (local time), computing the
// "now" day boundary once instead of allocating a Date per item per filter.
const grouped = computed(() => {
  const now = new Date()
  const y = now.getFullYear()
  const m = now.getMonth()
  const d = now.getDate()
  const today: Conversation[] = []
  const earlier: Conversation[] = []
  for (const c of filtered.value) {
    const dt = new Date(c.last_message_at || c.created_at)
    if (dt.getFullYear() === y && dt.getMonth() === m && dt.getDate() === d) {
      today.push(c)
    } else {
      earlier.push(c)
    }
  }
  return { today, earlier }
})

const todayItems = computed(() => grouped.value.today)
const earlierItems = computed(() => grouped.value.earlier)

const displayName = computed(() => {
  const u = authStore.user
  if (!u) return ''
  return u.username || u.email?.split('@')[0] || ''
})

const userInitial = computed(() => (displayName.value.slice(0, 1) || '?').toUpperCase())
const brandInitial = computed(() => t('chat.brandInitial'))

const subscriptionLabel = computed(() =>
  props.hasSubscription ? t('chat.subscriptionActive') : t('chat.noSubscription')
)
</script>
