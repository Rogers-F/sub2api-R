<template>
  <!-- Custom Home Content: Full Page Mode (admin-injected) -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Minimal warm landing (claude.ai aesthetic) -->
  <div
    v-else
    class="flex min-h-screen flex-col bg-paper-50 text-dust-800 dark:bg-ink-950 dark:text-pearl-100"
  >
    <!-- ==================== Nav ==================== -->
    <header
      class="sticky top-0 z-50 border-b hairline bg-paper-50/80 backdrop-blur-md dark:bg-ink-950/70"
    >
      <nav class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <a class="flex items-center gap-2.5" href="#top">
          <div
            class="flex h-8 w-8 items-center justify-center overflow-hidden rounded-lg border hairline bg-paper-0 dark:bg-ink-800"
          >
            <img v-if="siteLogo" :src="siteLogo" :alt="siteName" class="h-full w-full object-contain" />
            <span v-else class="font-display text-base text-coral-600 dark:text-coral-400">{{ siteName.slice(0, 1) }}</span>
          </div>
          <span class="font-display text-lg tracking-wide text-dust-900 dark:text-pearl-50">{{ siteName }}</span>
        </a>

        <div class="flex items-center gap-2 sm:gap-3">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener"
            class="hidden px-2 py-1 text-sm text-dust-600 transition-colors hover:text-dust-900 sm:inline dark:text-pearl-200 dark:hover:text-white"
          >
            {{ t('home.docs') }}
          </a>
          <LocaleSwitcher />
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-dust-500 transition-colors hover:bg-paper-100 hover:text-dust-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-coral-500/40 dark:text-pearl-300 dark:hover:bg-white/[0.06] dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" />
          </button>
          <router-link
            v-if="!isAuthenticated"
            to="/login"
            class="hidden px-2 py-1 text-sm text-dust-700 transition-colors hover:text-dust-900 sm:inline dark:text-pearl-200 dark:hover:text-white"
          >
            {{ t('home.login') }}
          </router-link>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-full bg-coral-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-coral-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-coral-500/50 focus-visible:ring-offset-2 focus-visible:ring-offset-paper-50 dark:focus-visible:ring-offset-ink-950"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- ==================== Hero ==================== -->
    <main id="top" class="relative flex-1 overflow-hidden">
      <!-- Warm flowing background (subtle) -->
      <div class="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        <div
          class="absolute left-1/2 top-[-12%] h-[820px] w-[1180px] -translate-x-1/2 rounded-full opacity-60 blur-3xl dark:opacity-30"
          style="background: radial-gradient(closest-side, rgba(232, 130, 107, 0.16), rgba(245, 244, 238, 0) 72%)"
        ></div>
        <div
          class="absolute bottom-[-22%] left-[10%] h-[620px] w-[760px] rounded-full opacity-50 blur-3xl dark:opacity-20"
          style="background: radial-gradient(closest-side, rgba(212, 182, 129, 0.12), rgba(245, 244, 238, 0) 70%)"
        ></div>
      </div>

      <div class="mx-auto flex max-w-3xl flex-col items-center px-6 py-24 text-center md:py-36">
        <h1
          class="font-display text-6xl font-medium leading-none tracking-tight text-dust-900 md:text-8xl dark:text-pearl-50"
        >
          {{ siteName }}
        </h1>
        <p class="mt-7 text-lg text-dust-600 md:text-xl dark:text-pearl-200">
          {{ t('home.heroMinimal.tagline') }}
        </p>

        <!-- Two CTA cards -->
        <div class="mt-12 grid w-full gap-4 sm:grid-cols-2">
          <router-link
            :to="isAuthenticated ? '/chat' : '/register'"
            class="group cursor-pointer rounded-2xl border border-coral-300/50 bg-coral-500/[0.05] p-6 text-left transition-colors hover:border-coral-400 hover:bg-coral-500/[0.09] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-coral-500/50 dark:border-coral-400/25 dark:bg-coral-400/[0.06] dark:hover:border-coral-400/50"
          >
            <div class="flex items-center justify-between">
              <span class="font-display text-xl text-coral-700 dark:text-coral-300">{{ t('home.cards.chat.title') }}</span>
              <Icon name="chatBubble" size="md" class="text-coral-500 dark:text-coral-400" />
            </div>
            <p class="mt-2 whitespace-pre-line text-sm leading-relaxed text-dust-600 dark:text-pearl-200">
              {{ t('home.cards.chat.desc') }}
            </p>
          </router-link>

          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="group cursor-pointer rounded-2xl border hairline bg-paper-0 p-6 text-left transition-colors hover:border-coral-400 hover:bg-paper-100/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-coral-500/50 dark:bg-ink-800 dark:hover:border-coral-400/40"
          >
            <div class="flex items-center justify-between">
              <span class="font-display text-xl text-dust-900 dark:text-pearl-50">{{ t('home.cards.console.title') }}</span>
              <Icon
                name="terminal"
                size="md"
                class="text-dust-400 transition-colors group-hover:text-coral-500 dark:text-pearl-300 dark:group-hover:text-coral-400"
              />
            </div>
            <p class="mt-2 whitespace-pre-line text-sm leading-relaxed text-dust-600 dark:text-pearl-200">
              {{ t('home.cards.console.desc') }}
            </p>
          </router-link>
        </div>
      </div>
    </main>

    <!-- ==================== Footer ==================== -->
    <footer class="border-t hairline bg-paper-100 dark:bg-ink-900">
      <div class="mx-auto max-w-6xl px-6 py-12">
        <div class="flex flex-col gap-8 md:flex-row md:justify-between">
          <!-- Brand -->
          <div class="max-w-xs">
            <div class="flex items-center gap-2.5">
              <div
                class="flex h-8 w-8 items-center justify-center overflow-hidden rounded-lg border hairline bg-paper-0 dark:bg-ink-800"
              >
                <img v-if="siteLogo" :src="siteLogo" :alt="siteName" class="h-full w-full object-contain" />
                <span v-else class="font-display text-base text-coral-600 dark:text-coral-400">{{ siteName.slice(0, 1) }}</span>
              </div>
              <span class="font-display text-lg text-dust-900 dark:text-pearl-50">{{ siteName }}</span>
            </div>
            <p class="mt-4 text-xs leading-relaxed text-dust-500 dark:text-pearl-300">{{ t('home.footer.tagline') }}</p>
          </div>

          <!-- Link columns -->
          <div class="grid grid-cols-2 gap-10">
            <div v-for="col in footerColumns" :key="col.key" class="space-y-3">
              <div class="text-[11px] font-medium uppercase tracking-[0.2em] text-dust-500 dark:text-pearl-300">
                {{ t(`home.footer.columns.${col.key}.title`) }}
              </div>
              <ul class="space-y-2">
                <li v-for="link in col.links" :key="link.labelKey">
                  <component
                    :is="link.to ? 'router-link' : 'a'"
                    :to="link.to"
                    :href="link.href"
                    :target="link.href ? '_blank' : undefined"
                    :rel="link.href ? 'noopener' : undefined"
                    class="text-xs text-dust-500 transition-colors hover:text-coral-600 dark:text-pearl-300 dark:hover:text-coral-400"
                  >
                    {{ t(link.labelKey) }}
                  </component>
                </li>
              </ul>
            </div>
          </div>
        </div>

        <!-- Bottom bar -->
        <div
          class="mt-10 flex flex-col items-center justify-between gap-3 border-t hairline pt-6 text-xs text-dust-400 sm:flex-row dark:text-pearl-400"
        >
          <div>© {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</div>
          <div class="flex items-center gap-4">
            <span class="font-mono">v{{ appVersion }}</span>
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener"
              class="transition-colors hover:text-dust-700 dark:hover:text-white"
            >
              {{ t('home.docs') }}
            </a>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

// === Public settings (admin-controlled) ===
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '星算')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const appVersion = (import.meta as ImportMeta & { env?: { VITE_APP_VERSION?: string } }).env?.VITE_APP_VERSION || '0.2.93'

// === Auth ===
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))

// === Theme ===
// 与 main.ts 共用 'theme' localStorage key,避免 home 与 dashboard 主题割裂
const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// === Misc ===
const currentYear = computed(() => new Date().getFullYear())

interface FooterLink {
  labelKey: string
  to?: string
  href?: string
}
interface FooterColumn {
  key: 'product' | 'resources'
  links: FooterLink[]
}

// Honest footer: only real destinations (no dead anchors). The resources column
// is shown only when a docs URL is configured.
const footerColumns = computed<FooterColumn[]>(() => {
  const cols: FooterColumn[] = [
    {
      key: 'product',
      links: [
        { labelKey: 'home.cards.chat.title', to: isAuthenticated.value ? '/chat' : '/register' },
        { labelKey: 'home.footer.columns.product.dashboard', to: dashboardPath.value },
        { labelKey: 'home.footer.columns.product.apiKeys', to: '/keys' },
        { labelKey: 'home.footer.columns.product.usage', to: '/usage' }
      ]
    }
  ]
  if (docUrl.value) {
    cols.push({
      key: 'resources',
      links: [{ labelKey: 'home.footer.columns.resources.docs', href: docUrl.value }]
    })
  }
  return cols
})

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
