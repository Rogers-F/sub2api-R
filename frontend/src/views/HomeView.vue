<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Starry Theme Home Page -->
  <div v-else class="relative min-h-screen overflow-hidden bg-[#0F172A]">
    <!-- Animated Starry Background -->
    <div class="fixed inset-0 z-0">
      <!-- Base gradient -->
      <div
        class="absolute inset-0 bg-gradient-to-b from-[#0F172A] via-[#1E293B] to-[#0F172A]"
      ></div>

      <!-- Aurora gradient overlay -->
      <div class="aurora-gradient"></div>

      <!-- Stars layers -->
      <div class="stars-layer stars-small"></div>
      <div class="stars-layer stars-medium"></div>
      <div class="stars-layer stars-large"></div>

      <!-- Shooting stars -->
      <div class="shooting-star"></div>
      <div class="shooting-star delay-1"></div>
      <div class="shooting-star delay-2"></div>

      <!-- Grid overlay -->
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(248,250,252,0.02)_1px,transparent_1px),linear-gradient(90deg,rgba(248,250,252,0.02)_1px,transparent_1px)] bg-[size:80px_80px] opacity-30"
      ></div>
    </div>

    <!-- Content -->
    <div class="relative z-10 flex min-h-screen flex-col">
      <!-- Header -->
      <header class="px-6 py-6">
        <nav class="mx-auto flex max-w-7xl items-center justify-between">
          <!-- Logo -->
          <div class="flex items-center gap-3">
            <div
              class="flex h-10 w-10 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-cyan-400/20 to-purple-600/20 shadow-lg shadow-cyan-500/20 backdrop-blur-sm ring-1 ring-white/10"
            >
              <img
                v-if="siteLogo"
                :src="siteLogo"
                alt="Logo"
                class="h-full w-full object-contain"
              />
              <span v-else class="text-lg font-bold text-cyan-400">{{ siteInitial }}</span>
            </div>
          </div>

          <!-- Nav Actions -->
          <div class="flex items-center gap-3">
            <!-- Language Switcher -->
            <LocaleSwitcher />

            <!-- Doc Link -->
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="rounded-lg p-2 text-gray-400 transition-all duration-200 hover:bg-white/5 hover:text-cyan-400"
              :title="t('home.viewDocs')"
            >
              <Icon name="book" size="md" />
            </a>

            <!-- Theme Toggle (hidden in starry theme) -->
            <button
              v-if="false"
              @click="toggleTheme"
              class="rounded-lg p-2 text-gray-400 transition-all duration-200 hover:bg-white/5 hover:text-cyan-400"
            >
              <Icon name="moon" size="md" />
            </button>

            <!-- Login / Dashboard Button -->
            <router-link
              v-if="isAuthenticated"
              :to="dashboardPath"
              class="group inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-cyan-500/10 to-purple-500/10 py-2 pl-2 pr-4 ring-1 ring-white/10 transition-all duration-300 hover:from-cyan-500/20 hover:to-purple-500/20 hover:ring-cyan-400/30"
            >
              <span
                class="flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-br from-cyan-400 to-purple-600 text-xs font-semibold text-white"
              >
                {{ userInitial }}
              </span>
              <span class="text-sm font-medium text-gray-200">{{ t('home.dashboard') }}</span>
              <svg
                class="h-4 w-4 text-gray-400 transition-transform group-hover:translate-x-0.5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
              </svg>
            </router-link>
            <router-link
              v-else
              to="/login"
              class="inline-flex items-center gap-2 rounded-full bg-white/5 px-4 py-2 text-sm font-medium text-gray-200 ring-1 ring-white/10 transition-all duration-300 hover:bg-white/10 hover:ring-cyan-400/30"
            >
              {{ t('home.login') }}
            </router-link>
          </div>
        </nav>
      </header>

      <!-- Main Content -->
      <main class="flex flex-1 items-center px-6 py-16">
        <div class="mx-auto w-full max-w-7xl">
          <!-- Hero Section - Centered -->
          <div class="mb-20 text-center">
            <!-- Main Heading -->
            <h1
              class="hero-heading mb-6 text-5xl font-bold leading-tight text-white sm:text-6xl lg:text-7xl"
            >
              {{ siteName }}
            </h1>

            <!-- Subtitle -->
            <p class="hero-subtitle mx-auto mb-12 max-w-3xl text-xl text-gray-300 sm:text-2xl">
              {{ siteSubtitle }}
            </p>

            <!-- CTA Button -->
            <div class="mb-16 flex justify-center gap-4">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="group cta-button inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-cyan-500 to-blue-600 px-8 py-4 text-base font-semibold text-white shadow-lg shadow-cyan-500/30 transition-all duration-300 hover:scale-105 hover:shadow-cyan-500/50"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="md" :stroke-width="2.5" />
              </router-link>
              <a
                v-if="docUrl"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-2 rounded-full bg-white/5 px-8 py-4 text-base font-semibold text-gray-200 ring-1 ring-white/10 transition-all duration-300 hover:bg-white/10 hover:ring-cyan-400/30"
              >
                {{ t('home.docs') }}
                <Icon name="book" size="md" />
              </a>
            </div>

            <!-- Feature Pills -->
            <div class="flex flex-wrap justify-center gap-3">
              <div
                class="feature-pill inline-flex items-center gap-2 rounded-full bg-white/5 px-4 py-2 text-sm font-medium text-gray-300 ring-1 ring-white/10 backdrop-blur-sm"
              >
                <div class="h-2 w-2 rounded-full bg-cyan-400 shadow-lg shadow-cyan-400/50"></div>
                {{ t('home.tags.subscriptionToApi') }}
              </div>
              <div
                class="feature-pill inline-flex items-center gap-2 rounded-full bg-white/5 px-4 py-2 text-sm font-medium text-gray-300 ring-1 ring-white/10 backdrop-blur-sm"
              >
                <div class="h-2 w-2 rounded-full bg-purple-400 shadow-lg shadow-purple-400/50"></div>
                {{ t('home.tags.stickySession') }}
              </div>
              <div
                class="feature-pill inline-flex items-center gap-2 rounded-full bg-white/5 px-4 py-2 text-sm font-medium text-gray-300 ring-1 ring-white/10 backdrop-blur-sm"
              >
                <div class="h-2 w-2 rounded-full bg-blue-400 shadow-lg shadow-blue-400/50"></div>
                {{ t('home.tags.realtimeBilling') }}
              </div>
            </div>
          </div>

          <!-- Features Grid -->
          <div class="mb-20 grid gap-6 md:grid-cols-2 max-w-4xl mx-auto">
            <!-- Feature 1: Unified Gateway -->
            <div
              class="feature-card group rounded-2xl bg-white/5 p-8 ring-1 ring-white/10 backdrop-blur-sm transition-all duration-300 hover:bg-white/10 hover:ring-cyan-400/30"
            >
              <div
                class="mb-5 flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-400/20 to-blue-600/20 text-cyan-400 ring-1 ring-cyan-400/30 transition-transform duration-300 group-hover:scale-110"
              >
                <Icon name="server" size="xl" :stroke-width="1.5" />
              </div>
              <h3 class="mb-3 text-xl font-semibold text-white">
                {{ t('home.features.unifiedGateway') }}
              </h3>
              <p class="text-gray-400">
                {{ t('home.features.unifiedGatewayDesc') }}
              </p>
            </div>

            <!-- Feature 2: Balance & Quota -->
            <div
              class="feature-card group rounded-2xl bg-white/5 p-8 ring-1 ring-white/10 backdrop-blur-sm transition-all duration-300 hover:bg-white/10 hover:ring-blue-400/30"
            >
              <div
                class="mb-5 flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-blue-400/20 to-indigo-600/20 text-blue-400 ring-1 ring-blue-400/30 transition-transform duration-300 group-hover:scale-110"
              >
                <Icon name="chart" size="xl" :stroke-width="1.5" />
              </div>
              <h3 class="mb-3 text-xl font-semibold text-white">
                {{ t('home.features.balanceQuota') }}
              </h3>
              <p class="text-gray-400">
                {{ t('home.features.balanceQuotaDesc') }}
              </p>
            </div>
          </div>

          <!-- Supported Providers -->
          <div class="text-center">
            <h2 class="mb-4 text-2xl font-semibold text-white">
              {{ t('home.providers.title') }}
            </h2>
            <p class="mb-8 text-gray-400">
              {{ t('home.providers.description') }}
            </p>

            <div class="flex flex-wrap justify-center gap-4">
              <!-- Claude -->
              <div
                class="provider-badge inline-flex items-center gap-3 rounded-xl bg-white/5 px-5 py-3 ring-1 ring-white/10 backdrop-blur-sm transition-all duration-300 hover:bg-white/10 hover:ring-orange-400/30"
              >
                <div
                  class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-orange-400 to-orange-600"
                >
                  <span class="text-sm font-bold text-white">C</span>
                </div>
                <span class="text-sm font-medium text-gray-200">{{
                  t('home.providers.claude')
                }}</span>
                <span
                  class="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs font-medium text-cyan-400"
                  >{{ t('home.providers.supported') }}</span
                >
              </div>

              <!-- GPT -->
              <div
                class="provider-badge inline-flex items-center gap-3 rounded-xl bg-white/5 px-5 py-3 ring-1 ring-white/10 backdrop-blur-sm transition-all duration-300 hover:bg-white/10 hover:ring-green-400/30"
              >
                <div
                  class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-green-500 to-green-700"
                >
                  <span class="text-sm font-bold text-white">G</span>
                </div>
                <span class="text-sm font-medium text-gray-200">GPT</span>
                <span
                  class="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs font-medium text-cyan-400"
                  >{{ t('home.providers.supported') }}</span
                >
              </div>

              <!-- More -->
              <div
                class="inline-flex items-center gap-3 rounded-xl bg-white/5 px-5 py-3 opacity-50 ring-1 ring-white/10 backdrop-blur-sm"
              >
                <div
                  class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-gray-500 to-gray-700"
                >
                  <span class="text-sm font-bold text-white">+</span>
                </div>
                <span class="text-sm font-medium text-gray-200">{{ t('home.providers.more') }}</span>
                <span class="rounded-full bg-gray-500/20 px-2 py-0.5 text-xs font-medium text-gray-400"
                  >{{ t('home.providers.soon') }}</span
                >
              </div>
            </div>
          </div>
        </div>
      </main>

      <!-- Footer -->
      <footer class="border-t border-white/5 px-6 py-8 backdrop-blur-sm">
        <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 sm:flex-row">
          <p class="text-sm text-gray-400">
            &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
          </p>
          <div class="flex items-center gap-6">
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm text-gray-400 transition-colors hover:text-cyan-400"
            >
              {{ t('home.docs') }}
            </a>
            <a
              :href="githubUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm text-gray-400 transition-colors hover:text-cyan-400"
            >
              GitHub
            </a>
          </div>
        </div>
      </footer>
    </div>
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

// Site settings
const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API'
)
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(
  () => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform'
)
const siteInitial = computed(() => siteName.value.charAt(0).toUpperCase())
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(true)

// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  // Force dark theme for starry mode
  isDark.value = true
  document.documentElement.classList.add('dark')

  // Check auth state
  authStore.checkAuth()

  // Load public settings
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
/* Import Google Fonts */
@import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;700&family=Space+Grotesk:wght@400;500;600;700&display=swap');

/* Font application */
.hero-heading {
  font-family: 'Space Grotesk', sans-serif;
  background: linear-gradient(to bottom, #ffffff, #94a3b8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-subtitle,
.feature-card,
.provider-badge {
  font-family: 'DM Sans', sans-serif;
}

/* Aurora gradient background */
.aurora-gradient {
  position: absolute;
  inset: 0;
  background: radial-gradient(
      ellipse 80% 50% at 50% -20%,
      rgba(0, 255, 255, 0.15),
      transparent
    ),
    radial-gradient(ellipse 60% 50% at 80% 50%, rgba(255, 0, 255, 0.1), transparent),
    radial-gradient(ellipse 60% 50% at 20% 80%, rgba(0, 102, 255, 0.1), transparent);
  animation: aurora-shift 20s ease-in-out infinite;
  opacity: 0.6;
}

@keyframes aurora-shift {
  0%,
  100% {
    opacity: 0.6;
    transform: scale(1) rotate(0deg);
  }
  50% {
    opacity: 0.8;
    transform: scale(1.1) rotate(5deg);
  }
}

/* Stars layers */
.stars-layer {
  position: absolute;
  inset: 0;
  background-repeat: repeat;
  animation: twinkle linear infinite;
}

.stars-small {
  background-image: radial-gradient(2px 2px at 20% 30%, rgba(255, 255, 255, 0.8), transparent),
    radial-gradient(2px 2px at 60% 70%, rgba(255, 255, 255, 0.6), transparent),
    radial-gradient(1px 1px at 50% 50%, rgba(255, 255, 255, 0.4), transparent),
    radial-gradient(1px 1px at 80% 10%, rgba(255, 255, 255, 0.5), transparent),
    radial-gradient(2px 2px at 90% 60%, rgba(255, 255, 255, 0.7), transparent),
    radial-gradient(1px 1px at 33% 80%, rgba(255, 255, 255, 0.3), transparent);
  background-size: 200% 200%;
  animation-duration: 8s;
}

.stars-medium {
  background-image: radial-gradient(3px 3px at 30% 20%, rgba(0, 255, 255, 0.6), transparent),
    radial-gradient(2px 2px at 70% 80%, rgba(147, 51, 234, 0.5), transparent),
    radial-gradient(3px 3px at 40% 60%, rgba(59, 130, 246, 0.6), transparent),
    radial-gradient(2px 2px at 85% 40%, rgba(255, 255, 255, 0.7), transparent);
  background-size: 300% 300%;
  animation-duration: 12s;
}

.stars-large {
  background-image: radial-gradient(4px 4px at 25% 50%, rgba(0, 255, 255, 0.8), transparent),
    radial-gradient(3px 3px at 75% 25%, rgba(168, 85, 247, 0.7), transparent),
    radial-gradient(4px 4px at 50% 75%, rgba(59, 130, 246, 0.8), transparent);
  background-size: 400% 400%;
  animation-duration: 16s;
}

@keyframes twinkle {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Shooting stars */
.shooting-star {
  position: absolute;
  top: -2px;
  left: 50%;
  width: 2px;
  height: 2px;
  background: linear-gradient(to right, rgba(255, 255, 255, 1), transparent);
  border-radius: 50%;
  box-shadow: 0 0 10px 2px rgba(255, 255, 255, 0.5);
  animation: shoot 3s ease-in-out infinite;
  opacity: 0;
}

.shooting-star.delay-1 {
  animation-delay: 1s;
  left: 30%;
}

.shooting-star.delay-2 {
  animation-delay: 2s;
  left: 70%;
}

@keyframes shoot {
  0% {
    opacity: 0;
    transform: translateX(-100px) translateY(0) rotate(-45deg);
  }
  10% {
    opacity: 1;
  }
  90% {
    opacity: 1;
  }
  100% {
    opacity: 0;
    transform: translateX(300px) translateY(200px) rotate(-45deg);
  }
}

/* Hover effects */
.feature-card:hover,
.provider-badge:hover {
  transform: translateY(-4px);
}

.cta-button:hover {
  box-shadow:
    0 20px 25px -5px rgba(6, 182, 212, 0.3),
    0 10px 10px -5px rgba(6, 182, 212, 0.2);
}

.feature-pill {
  animation: float 3s ease-in-out infinite;
}

.feature-pill:nth-child(2) {
  animation-delay: 0.5s;
}

.feature-pill:nth-child(3) {
  animation-delay: 1s;
}

@keyframes float {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-5px);
  }
}

/* Smooth transitions */
* {
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
}

/* Responsive */
@media (max-width: 768px) {
  .hero-heading {
    font-size: 2.5rem;
  }

  .hero-subtitle {
    font-size: 1.125rem;
  }
}

/* Accessibility */
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}

/* Focus states */
a:focus-visible,
button:focus-visible {
  outline: 2px solid rgba(6, 182, 212, 0.5);
  outline-offset: 2px;
}

/* Cursor pointer */
a,
button,
.cursor-pointer {
  cursor: pointer;
}
</style>
