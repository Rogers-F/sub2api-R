<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- ==================== Light Mode (Warm Minimalist) ==================== -->
  <div v-else-if="!isDark" class="min-h-screen bg-warm-50">
    <!-- Header -->
    <header class="sticky top-0 z-50 border-b border-warm-200 bg-warm-50/80 backdrop-blur-sm">
      <nav class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <!-- Logo -->
        <div class="flex items-center gap-3">
          <div class="flex h-9 w-9 items-center justify-center overflow-hidden rounded-lg bg-clay-500">
            <img v-if="siteLogo" :src="siteLogo" alt="Logo" class="h-full w-full object-contain" />
            <span v-else class="text-base font-bold text-white">{{ siteInitial }}</span>
          </div>
          <span class="text-lg font-semibold tracking-tight text-warm-900">{{ siteName }}</span>
        </div>

        <!-- Nav Links -->
        <div class="hidden items-center gap-6 md:flex">
          <a href="#home" class="text-sm font-medium text-warm-700 transition-colors hover:text-warm-900">{{ t('home.nav.home') }}</a>
          <a href="#pricing" class="text-sm font-medium text-warm-700 transition-colors hover:text-warm-900">{{ t('home.nav.pricing') }}</a>
          <a v-if="docUrl" :href="docUrl" target="_blank" class="text-sm font-medium text-warm-700 transition-colors hover:text-warm-900">{{ t('home.docs') }}</a>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <LocaleSwitcher />
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-warm-700 transition-colors hover:bg-warm-100"
            :title="t('home.switchToDark')"
          >
            <Icon name="moon" size="md" />
          </button>
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-2 rounded-lg bg-warm-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-warm-800"
          >
            {{ t('home.dashboard') }}
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center gap-2 rounded-lg bg-warm-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-warm-800"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Hero Section -->
    <section id="home" class="px-6 py-20 md:py-28">
      <div class="mx-auto max-w-4xl text-center">
        <h1 class="mb-6 text-4xl font-bold tracking-tight text-warm-900 sm:text-5xl lg:text-6xl">
          {{ siteName }} {{ t('home.hero.tagline') }}
        </h1>
        <p class="mx-auto mb-10 max-w-2xl text-lg leading-relaxed text-warm-700">
          {{ t('home.hero.description') }}
        </p>
        <div class="mb-12 flex flex-wrap justify-center gap-4">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex items-center gap-2 rounded-lg bg-warm-900 px-6 py-3 text-base font-semibold text-white shadow-warm transition-all hover:bg-warm-800 hover:shadow-warm-lg"
          >
            {{ t('home.hero.cta') }}
            <Icon name="arrowRight" size="md" />
          </router-link>
        </div>
        <!-- Feature Tags -->
        <div class="flex flex-wrap justify-center gap-3">
          <span class="inline-flex items-center gap-2 rounded-full border border-warm-200 bg-white px-4 py-2 text-sm font-medium text-warm-700 shadow-warm-sm">
            <span class="h-2 w-2 rounded-full bg-primary-500"></span>
            {{ t('home.hero.tags.codeGen') }}
          </span>
          <span class="inline-flex items-center gap-2 rounded-full border border-warm-200 bg-white px-4 py-2 text-sm font-medium text-warm-700 shadow-warm-sm">
            <span class="h-2 w-2 rounded-full bg-clay-500"></span>
            {{ t('home.hero.tags.codeUnderstand') }}
          </span>
          <span class="inline-flex items-center gap-2 rounded-full border border-warm-200 bg-white px-4 py-2 text-sm font-medium text-warm-700 shadow-warm-sm">
            <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
            {{ t('home.hero.tags.docGen') }}
          </span>
        </div>
      </div>
    </section>

    <!-- Principles Section -->
    <section class="border-t border-warm-200 bg-white px-6 py-20">
      <div class="mx-auto max-w-6xl">
        <div class="mb-4 text-center text-sm font-medium uppercase tracking-wider text-warm-500">{{ t('home.principles.label') }}</div>
        <h2 class="mb-4 text-center text-3xl font-bold text-warm-900 sm:text-4xl">
          {{ t('home.principles.title') }}
        </h2>
        <p class="mx-auto mb-16 max-w-3xl text-center text-warm-700">
          {{ t('home.principles.description') }}
        </p>

        <!-- Feature Cards -->
        <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          <div class="rounded-2xl border border-warm-200 bg-warm-50 p-6 transition-all hover:border-warm-300 hover:shadow-warm">
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-primary-100 text-primary-600">
              <Icon name="cog" size="lg" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-warm-900">{{ t('home.principles.features.simpleConfig.title') }}</h3>
            <p class="text-sm leading-relaxed text-warm-700">{{ t('home.principles.features.simpleConfig.desc') }}</p>
          </div>
          <div class="rounded-2xl border border-warm-200 bg-warm-50 p-6 transition-all hover:border-warm-300 hover:shadow-warm">
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-100 text-emerald-600">
              <Icon name="chart" size="lg" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-warm-900">{{ t('home.principles.features.accurateBilling.title') }}</h3>
            <p class="text-sm leading-relaxed text-warm-700">{{ t('home.principles.features.accurateBilling.desc') }}</p>
          </div>
          <div class="rounded-2xl border border-warm-200 bg-warm-50 p-6 transition-all hover:border-warm-300 hover:shadow-warm">
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-blue-100 text-blue-600">
              <Icon name="eye" size="lg" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-warm-900">{{ t('home.principles.features.transparent.title') }}</h3>
            <p class="text-sm leading-relaxed text-warm-700">{{ t('home.principles.features.transparent.desc') }}</p>
          </div>
          <div class="rounded-2xl border border-warm-200 bg-warm-50 p-6 transition-all hover:border-warm-300 hover:shadow-warm">
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-amber-100 text-amber-600">
              <Icon name="bolt" size="lg" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-warm-900">{{ t('home.principles.features.devFocused.title') }}</h3>
            <p class="text-sm leading-relaxed text-warm-700">{{ t('home.principles.features.devFocused.desc') }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Metrics Section -->
    <section class="border-t border-warm-200 bg-warm-50 px-6 py-16">
      <div class="mx-auto max-w-6xl">
        <div class="mb-8 text-center text-xs font-semibold uppercase tracking-widest text-warm-500">CODING METRICS</div>
        <div class="grid gap-6 md:grid-cols-3">
          <div class="rounded-2xl border border-warm-200 bg-white p-6 text-center shadow-warm-sm">
            <div class="mb-2 text-sm text-warm-600">{{ t('home.metrics.latency.label') }}</div>
            <div class="text-3xl font-bold text-warm-900">&lt; 2000 <span class="text-lg font-normal text-warm-500">ms</span></div>
          </div>
          <div class="rounded-2xl border border-warm-200 bg-white p-6 text-center shadow-warm-sm">
            <div class="mb-2 text-sm text-warm-600">{{ t('home.metrics.cacheSave.label') }}</div>
            <div class="text-3xl font-bold text-emerald-600">&gt; 60<span class="text-lg font-normal text-warm-500">%</span></div>
          </div>
          <div class="rounded-2xl border border-warm-200 bg-white p-6 text-center shadow-warm-sm">
            <div class="mb-2 text-sm text-warm-600">{{ t('home.metrics.rate.label') }}</div>
            <div class="text-3xl font-bold text-warm-900">{{ t('home.metrics.rate.value') }}</div>
          </div>
        </div>
        <p class="mt-6 text-center text-xs text-warm-500">{{ t('home.metrics.note') }}</p>
      </div>
    </section>

    <!-- Why Choose Section -->
    <section class="border-t border-warm-200 bg-white px-6 py-20">
      <div class="mx-auto max-w-6xl">
        <h2 class="mb-4 text-center text-3xl font-bold text-warm-900">{{ t('home.whyChoose.title') }}</h2>
        <p class="mx-auto mb-12 max-w-2xl text-center text-warm-700">{{ t('home.whyChoose.description') }}</p>
        <div class="grid gap-8 md:grid-cols-3">
          <div class="text-center">
            <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-100 text-primary-600">
              <Icon name="terminal" size="xl" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-warm-900">{{ t('home.whyChoose.features.ideSupport.title') }}</h3>
            <p class="text-sm text-warm-700">{{ t('home.whyChoose.features.ideSupport.desc') }}</p>
          </div>
          <div class="text-center">
            <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-100 text-emerald-600">
              <Icon name="shield" size="xl" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-warm-900">{{ t('home.whyChoose.features.stable.title') }}</h3>
            <p class="text-sm text-warm-700">{{ t('home.whyChoose.features.stable.desc') }}</p>
          </div>
          <div class="text-center">
            <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-blue-100 text-blue-600">
              <Icon name="document" size="xl" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-warm-900">{{ t('home.whyChoose.features.traceable.title') }}</h3>
            <p class="text-sm text-warm-700">{{ t('home.whyChoose.features.traceable.desc') }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Model Pricing Section -->
    <section id="pricing" class="border-t border-warm-200 bg-warm-50 px-6 py-20">
      <div class="mx-auto max-w-6xl">
        <h2 class="mb-4 text-center text-3xl font-bold text-warm-900">{{ t('home.pricing.title') }}</h2>
        <p class="mx-auto mb-12 max-w-3xl text-center text-warm-700">{{ t('home.pricing.description') }}</p>

        <!-- Pricing Cards -->
        <div class="grid gap-6 lg:grid-cols-3">
          <!-- Claude Opus 4.6 -->
          <div class="overflow-hidden rounded-2xl border border-warm-200 bg-white shadow-warm-sm transition-all hover:shadow-warm">
            <div class="border-b border-warm-100 bg-gradient-to-r from-clay-500/10 to-clay-600/5 px-6 py-4">
              <div class="mb-1 flex items-center gap-2">
                <span class="rounded bg-clay-100 px-2 py-0.5 text-xs font-medium text-clay-600">Anthropic</span>
                <span class="rounded bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700">{{ t('home.pricing.tags.powerful') }}</span>
              </div>
              <h3 class="text-xl font-bold text-warm-900">Claude Opus 4.6</h3>
              <p class="mt-1 text-sm text-warm-600">{{ t('home.pricing.models.opus.desc') }}</p>
            </div>
            <div class="p-6">
              <div class="mb-4 grid grid-cols-2 gap-4">
                <div>
                  <div class="text-xs text-warm-500">{{ t('home.pricing.input') }}</div>
                  <div class="text-2xl font-bold text-warm-900">$5<span class="text-sm font-normal text-warm-500"> / MTok</span></div>
                </div>
                <div>
                  <div class="text-xs text-warm-500">{{ t('home.pricing.output') }}</div>
                  <div class="text-2xl font-bold text-warm-900">$25<span class="text-sm font-normal text-warm-500"> / MTok</span></div>
                </div>
              </div>
              <div class="rounded-lg bg-warm-50 p-3">
                <div class="mb-2 text-xs font-medium text-warm-600">{{ t('home.pricing.cachePrice') }}</div>
                <div class="flex justify-between text-sm">
                  <span class="text-warm-600">{{ t('home.pricing.cacheWrite') }}</span>
                  <span class="font-semibold text-warm-900">$6.25</span>
                </div>
                <div class="flex justify-between text-sm">
                  <span class="text-warm-600">{{ t('home.pricing.cacheRead') }}</span>
                  <span class="font-semibold text-warm-900">$0.50</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Claude Sonnet 4.5 -->
          <div class="overflow-hidden rounded-2xl border-2 border-primary-500 bg-white shadow-warm transition-all hover:shadow-warm-lg">
            <div class="border-b border-warm-100 bg-gradient-to-r from-primary-500/10 to-primary-600/5 px-6 py-4">
              <div class="mb-1 flex items-center gap-2">
                <span class="rounded bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700">Anthropic</span>
                <span class="rounded bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700">{{ t('home.pricing.tags.bestValue') }}</span>
              </div>
              <h3 class="text-xl font-bold text-warm-900">Claude Sonnet 4.5</h3>
              <p class="mt-1 text-sm text-warm-600">{{ t('home.pricing.models.sonnet.desc') }}</p>
            </div>
            <div class="p-6">
              <div class="mb-4 grid grid-cols-2 gap-4">
                <div>
                  <div class="text-xs text-warm-500">{{ t('home.pricing.input') }}</div>
                  <div class="text-2xl font-bold text-warm-900">$3<span class="text-sm font-normal text-warm-500"> / MTok</span></div>
                </div>
                <div>
                  <div class="text-xs text-warm-500">{{ t('home.pricing.output') }}</div>
                  <div class="text-2xl font-bold text-warm-900">$15<span class="text-sm font-normal text-warm-500"> / MTok</span></div>
                </div>
              </div>
              <div class="rounded-lg bg-warm-50 p-3">
                <div class="mb-2 text-xs font-medium text-warm-600">{{ t('home.pricing.cachePrice') }}</div>
                <div class="flex justify-between text-sm">
                  <span class="text-warm-600">{{ t('home.pricing.cacheWrite') }}</span>
                  <span class="font-semibold text-warm-900">$3.75</span>
                </div>
                <div class="flex justify-between text-sm">
                  <span class="text-warm-600">{{ t('home.pricing.cacheRead') }}</span>
                  <span class="font-semibold text-warm-900">$0.30</span>
                </div>
              </div>
            </div>
          </div>

          <!-- GPT-5.2-Codex -->
          <div class="overflow-hidden rounded-2xl border border-warm-200 bg-white shadow-warm-sm transition-all hover:shadow-warm">
            <div class="border-b border-warm-100 bg-gradient-to-r from-emerald-500/10 to-emerald-600/5 px-6 py-4">
              <div class="mb-1 flex items-center gap-2">
                <span class="rounded bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700">OpenAI</span>
                <span class="rounded bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">{{ t('home.pricing.tags.efficient') }}</span>
              </div>
              <h3 class="text-xl font-bold text-warm-900">GPT-5.2 Codex</h3>
              <p class="mt-1 text-sm text-warm-600">{{ t('home.pricing.models.gpt.desc') }}</p>
            </div>
            <div class="p-6">
              <div class="mb-4 grid grid-cols-2 gap-4">
                <div>
                  <div class="text-xs text-warm-500">{{ t('home.pricing.input') }}</div>
                  <div class="text-2xl font-bold text-warm-900">$1.75<span class="text-sm font-normal text-warm-500"> / MTok</span></div>
                </div>
                <div>
                  <div class="text-xs text-warm-500">{{ t('home.pricing.output') }}</div>
                  <div class="text-2xl font-bold text-warm-900">$14<span class="text-sm font-normal text-warm-500"> / MTok</span></div>
                </div>
              </div>
              <div class="rounded-lg bg-warm-50 p-3">
                <div class="mb-2 text-xs font-medium text-warm-600">{{ t('home.pricing.cachePrice') }}</div>
                <div class="flex justify-between text-sm">
                  <span class="text-warm-600">{{ t('home.pricing.cacheWrite') }}</span>
                  <span class="font-semibold text-warm-500">-</span>
                </div>
                <div class="flex justify-between text-sm">
                  <span class="text-warm-600">{{ t('home.pricing.cacheRead') }}</span>
                  <span class="font-semibold text-warm-900">$0.175</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <p class="mt-8 text-center text-xs text-warm-500">{{ t('home.pricing.note') }}</p>

        <!-- CTA -->
        <div class="mt-10 text-center">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex items-center gap-2 rounded-lg bg-warm-900 px-8 py-3 text-base font-semibold text-white shadow-warm transition-all hover:bg-warm-800 hover:shadow-warm-lg"
          >
            {{ t('home.pricing.cta') }}
            <Icon name="arrowRight" size="md" />
          </router-link>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="border-t border-warm-200 bg-white px-6 py-8">
      <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 sm:flex-row">
        <p class="text-sm text-warm-600">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div v-if="docUrl" class="flex items-center gap-6">
          <a :href="docUrl" target="_blank" class="text-sm text-warm-600 transition-colors hover:text-warm-900">{{ t('home.docs') }}</a>
        </div>
      </div>
    </footer>
  </div>

  <!-- ==================== Dark Mode (Starry Theme) ==================== -->
  <div v-else class="relative min-h-screen overflow-hidden bg-[#0F172A]">
    <!-- Animated Starry Background -->
    <div class="fixed inset-0 z-0">
      <div class="absolute inset-0 bg-gradient-to-b from-[#0F172A] via-[#1E293B] to-[#0F172A]"></div>
      <div class="aurora-gradient"></div>
      <div class="stars-layer stars-small"></div>
      <div class="stars-layer stars-medium"></div>
      <div class="stars-layer stars-large"></div>
      <div class="shooting-star"></div>
      <div class="shooting-star delay-1"></div>
      <div class="shooting-star delay-2"></div>
    </div>

    <!-- Dark Mode Content -->
    <div class="relative z-10 flex min-h-screen flex-col">
      <!-- Header -->
      <header class="px-6 py-6">
        <nav class="mx-auto flex max-w-7xl items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-cyan-400/20 to-purple-600/20 shadow-lg shadow-cyan-500/20 backdrop-blur-sm ring-1 ring-white/10">
              <img v-if="siteLogo" :src="siteLogo" alt="Logo" class="h-full w-full object-contain" />
              <span v-else class="text-lg font-bold text-cyan-400">{{ siteInitial }}</span>
            </div>
            <span class="text-lg font-semibold text-white">{{ siteName }}</span>
          </div>
          <div class="flex items-center gap-3">
            <LocaleSwitcher />
            <a v-if="docUrl" :href="docUrl" target="_blank" class="rounded-lg p-2 text-gray-400 transition-all hover:bg-white/5 hover:text-cyan-400">
              <Icon name="book" size="md" />
            </a>
            <button @click="toggleTheme" class="rounded-lg p-2 text-gray-400 transition-all hover:bg-white/5 hover:text-cyan-400" :title="t('home.switchToLight')">
              <Icon name="sun" size="md" />
            </button>
            <router-link v-if="isAuthenticated" :to="dashboardPath" class="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-cyan-500/10 to-purple-500/10 py-2 pl-2 pr-4 ring-1 ring-white/10 transition-all hover:ring-cyan-400/30">
              <span class="flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-br from-cyan-400 to-purple-600 text-xs font-semibold text-white">{{ userInitial }}</span>
              <span class="text-sm font-medium text-gray-200">{{ t('home.dashboard') }}</span>
            </router-link>
            <router-link v-else to="/login" class="inline-flex items-center gap-2 rounded-full bg-white/5 px-4 py-2 text-sm font-medium text-gray-200 ring-1 ring-white/10 transition-all hover:bg-white/10 hover:ring-cyan-400/30">
              {{ t('home.login') }}
            </router-link>
          </div>
        </nav>
      </header>

      <!-- Dark Mode Hero -->
      <section id="home-dark" class="px-6 py-20 md:py-28">
        <div class="mx-auto max-w-4xl text-center">
          <h1 class="hero-heading mb-6 text-4xl font-bold leading-tight sm:text-5xl lg:text-6xl">
            {{ siteName }} {{ t('home.hero.tagline') }}
          </h1>
          <p class="mx-auto mb-10 max-w-2xl text-lg leading-relaxed text-gray-300">
            {{ t('home.hero.description') }}
          </p>
          <div class="mb-12 flex flex-wrap justify-center gap-4">
            <router-link
              :to="isAuthenticated ? dashboardPath : '/login'"
              class="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-cyan-500 to-blue-600 px-8 py-3 text-base font-semibold text-white shadow-lg shadow-cyan-500/30 transition-all hover:scale-105 hover:shadow-cyan-500/50"
            >
              {{ t('home.hero.cta') }}
              <Icon name="arrowRight" size="md" />
            </router-link>
          </div>
          <!-- Feature Tags -->
          <div class="flex flex-wrap justify-center gap-3">
            <span class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-gray-300 backdrop-blur-sm">
              <span class="h-2 w-2 rounded-full bg-cyan-400"></span>
              {{ t('home.hero.tags.codeGen') }}
            </span>
            <span class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-gray-300 backdrop-blur-sm">
              <span class="h-2 w-2 rounded-full bg-purple-400"></span>
              {{ t('home.hero.tags.codeUnderstand') }}
            </span>
            <span class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-gray-300 backdrop-blur-sm">
              <span class="h-2 w-2 rounded-full bg-emerald-400"></span>
              {{ t('home.hero.tags.docGen') }}
            </span>
          </div>
        </div>
      </section>

      <!-- Principles Section (Dark) -->
      <section class="border-t border-white/5 px-6 py-20">
        <div class="mx-auto max-w-6xl">
          <div class="mb-4 text-center text-sm font-medium uppercase tracking-wider text-gray-500">{{ t('home.principles.label') }}</div>
          <h2 class="mb-4 text-center text-3xl font-bold text-white sm:text-4xl">
            {{ t('home.principles.title') }}
          </h2>
          <p class="mx-auto mb-16 max-w-3xl text-center text-gray-400">
            {{ t('home.principles.description') }}
          </p>
          <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
            <div class="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm transition-all hover:border-cyan-400/20 hover:bg-white/10">
              <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-cyan-500/10 text-cyan-400">
                <Icon name="cog" size="lg" />
              </div>
              <h3 class="mb-2 text-lg font-semibold text-white">{{ t('home.principles.features.simpleConfig.title') }}</h3>
              <p class="text-sm leading-relaxed text-gray-400">{{ t('home.principles.features.simpleConfig.desc') }}</p>
            </div>
            <div class="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm transition-all hover:border-emerald-400/20 hover:bg-white/10">
              <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400">
                <Icon name="chart" size="lg" />
              </div>
              <h3 class="mb-2 text-lg font-semibold text-white">{{ t('home.principles.features.accurateBilling.title') }}</h3>
              <p class="text-sm leading-relaxed text-gray-400">{{ t('home.principles.features.accurateBilling.desc') }}</p>
            </div>
            <div class="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm transition-all hover:border-blue-400/20 hover:bg-white/10">
              <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400">
                <Icon name="eye" size="lg" />
              </div>
              <h3 class="mb-2 text-lg font-semibold text-white">{{ t('home.principles.features.transparent.title') }}</h3>
              <p class="text-sm leading-relaxed text-gray-400">{{ t('home.principles.features.transparent.desc') }}</p>
            </div>
            <div class="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm transition-all hover:border-amber-400/20 hover:bg-white/10">
              <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400">
                <Icon name="bolt" size="lg" />
              </div>
              <h3 class="mb-2 text-lg font-semibold text-white">{{ t('home.principles.features.devFocused.title') }}</h3>
              <p class="text-sm leading-relaxed text-gray-400">{{ t('home.principles.features.devFocused.desc') }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Metrics Section (Dark) -->
      <section class="border-t border-white/5 px-6 py-16">
        <div class="mx-auto max-w-6xl">
          <div class="mb-8 text-center text-xs font-semibold uppercase tracking-widest text-gray-500">CODING METRICS</div>
          <div class="grid gap-6 md:grid-cols-3">
            <div class="rounded-2xl border border-white/10 bg-white/5 p-6 text-center backdrop-blur-sm">
              <div class="mb-2 text-sm text-gray-400">{{ t('home.metrics.latency.label') }}</div>
              <div class="text-3xl font-bold text-white">&lt; 2000 <span class="text-lg font-normal text-gray-500">ms</span></div>
            </div>
            <div class="rounded-2xl border border-white/10 bg-white/5 p-6 text-center backdrop-blur-sm">
              <div class="mb-2 text-sm text-gray-400">{{ t('home.metrics.cacheSave.label') }}</div>
              <div class="text-3xl font-bold text-emerald-400">&gt; 60<span class="text-lg font-normal text-gray-500">%</span></div>
            </div>
            <div class="rounded-2xl border border-white/10 bg-white/5 p-6 text-center backdrop-blur-sm">
              <div class="mb-2 text-sm text-gray-400">{{ t('home.metrics.rate.label') }}</div>
              <div class="text-3xl font-bold text-white">{{ t('home.metrics.rate.value') }}</div>
            </div>
          </div>
          <p class="mt-6 text-center text-xs text-gray-500">{{ t('home.metrics.note') }}</p>
        </div>
      </section>

      <!-- Why Choose Section (Dark) -->
      <section class="border-t border-white/5 px-6 py-20">
        <div class="mx-auto max-w-6xl">
          <h2 class="mb-4 text-center text-3xl font-bold text-white">{{ t('home.whyChoose.title') }}</h2>
          <p class="mx-auto mb-12 max-w-2xl text-center text-gray-400">{{ t('home.whyChoose.description') }}</p>
          <div class="grid gap-8 md:grid-cols-3">
            <div class="text-center">
              <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-cyan-500/10 text-cyan-400">
                <Icon name="terminal" size="xl" />
              </div>
              <h3 class="mb-2 text-lg font-semibold text-white">{{ t('home.whyChoose.features.ideSupport.title') }}</h3>
              <p class="text-sm text-gray-400">{{ t('home.whyChoose.features.ideSupport.desc') }}</p>
            </div>
            <div class="text-center">
              <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-400">
                <Icon name="shield" size="xl" />
              </div>
              <h3 class="mb-2 text-lg font-semibold text-white">{{ t('home.whyChoose.features.stable.title') }}</h3>
              <p class="text-sm text-gray-400">{{ t('home.whyChoose.features.stable.desc') }}</p>
            </div>
            <div class="text-center">
              <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-blue-500/10 text-blue-400">
                <Icon name="document" size="xl" />
              </div>
              <h3 class="mb-2 text-lg font-semibold text-white">{{ t('home.whyChoose.features.traceable.title') }}</h3>
              <p class="text-sm text-gray-400">{{ t('home.whyChoose.features.traceable.desc') }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Model Pricing Section (Dark) -->
      <section id="pricing-dark" class="border-t border-white/5 px-6 py-20">
        <div class="mx-auto max-w-6xl">
          <h2 class="mb-4 text-center text-3xl font-bold text-white">{{ t('home.pricing.title') }}</h2>
          <p class="mx-auto mb-12 max-w-3xl text-center text-gray-400">{{ t('home.pricing.description') }}</p>

          <div class="grid gap-6 lg:grid-cols-3">
            <!-- Claude Opus 4.6 -->
            <div class="overflow-hidden rounded-2xl border border-white/10 bg-white/5 backdrop-blur-sm transition-all hover:border-white/20 hover:bg-white/10">
              <div class="border-b border-white/5 bg-gradient-to-r from-orange-500/10 to-orange-600/5 px-6 py-4">
                <div class="mb-1 flex items-center gap-2">
                  <span class="rounded bg-orange-500/20 px-2 py-0.5 text-xs font-medium text-orange-300">Anthropic</span>
                  <span class="rounded bg-amber-500/20 px-2 py-0.5 text-xs font-medium text-amber-300">{{ t('home.pricing.tags.powerful') }}</span>
                </div>
                <h3 class="text-xl font-bold text-white">Claude Opus 4.6</h3>
                <p class="mt-1 text-sm text-gray-400">{{ t('home.pricing.models.opus.desc') }}</p>
              </div>
              <div class="p-6">
                <div class="mb-4 grid grid-cols-2 gap-4">
                  <div>
                    <div class="text-xs text-gray-500">{{ t('home.pricing.input') }}</div>
                    <div class="text-2xl font-bold text-white">$5<span class="text-sm font-normal text-gray-500"> / MTok</span></div>
                  </div>
                  <div>
                    <div class="text-xs text-gray-500">{{ t('home.pricing.output') }}</div>
                    <div class="text-2xl font-bold text-white">$25<span class="text-sm font-normal text-gray-500"> / MTok</span></div>
                  </div>
                </div>
                <div class="rounded-lg bg-white/5 p-3">
                  <div class="mb-2 text-xs font-medium text-gray-400">{{ t('home.pricing.cachePrice') }}</div>
                  <div class="flex justify-between text-sm">
                    <span class="text-gray-400">{{ t('home.pricing.cacheWrite') }}</span>
                    <span class="font-semibold text-white">$6.25</span>
                  </div>
                  <div class="flex justify-between text-sm">
                    <span class="text-gray-400">{{ t('home.pricing.cacheRead') }}</span>
                    <span class="font-semibold text-white">$0.50</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Claude Sonnet 4.5 -->
            <div class="overflow-hidden rounded-2xl border-2 border-cyan-500/40 bg-white/5 backdrop-blur-sm transition-all hover:border-cyan-400/60 hover:bg-white/10">
              <div class="border-b border-white/5 bg-gradient-to-r from-cyan-500/10 to-blue-600/5 px-6 py-4">
                <div class="mb-1 flex items-center gap-2">
                  <span class="rounded bg-cyan-500/20 px-2 py-0.5 text-xs font-medium text-cyan-300">Anthropic</span>
                  <span class="rounded bg-emerald-500/20 px-2 py-0.5 text-xs font-medium text-emerald-300">{{ t('home.pricing.tags.bestValue') }}</span>
                </div>
                <h3 class="text-xl font-bold text-white">Claude Sonnet 4.5</h3>
                <p class="mt-1 text-sm text-gray-400">{{ t('home.pricing.models.sonnet.desc') }}</p>
              </div>
              <div class="p-6">
                <div class="mb-4 grid grid-cols-2 gap-4">
                  <div>
                    <div class="text-xs text-gray-500">{{ t('home.pricing.input') }}</div>
                    <div class="text-2xl font-bold text-white">$3<span class="text-sm font-normal text-gray-500"> / MTok</span></div>
                  </div>
                  <div>
                    <div class="text-xs text-gray-500">{{ t('home.pricing.output') }}</div>
                    <div class="text-2xl font-bold text-white">$15<span class="text-sm font-normal text-gray-500"> / MTok</span></div>
                  </div>
                </div>
                <div class="rounded-lg bg-white/5 p-3">
                  <div class="mb-2 text-xs font-medium text-gray-400">{{ t('home.pricing.cachePrice') }}</div>
                  <div class="flex justify-between text-sm">
                    <span class="text-gray-400">{{ t('home.pricing.cacheWrite') }}</span>
                    <span class="font-semibold text-white">$3.75</span>
                  </div>
                  <div class="flex justify-between text-sm">
                    <span class="text-gray-400">{{ t('home.pricing.cacheRead') }}</span>
                    <span class="font-semibold text-white">$0.30</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- GPT-5.2-Codex -->
            <div class="overflow-hidden rounded-2xl border border-white/10 bg-white/5 backdrop-blur-sm transition-all hover:border-white/20 hover:bg-white/10">
              <div class="border-b border-white/5 bg-gradient-to-r from-emerald-500/10 to-emerald-600/5 px-6 py-4">
                <div class="mb-1 flex items-center gap-2">
                  <span class="rounded bg-emerald-500/20 px-2 py-0.5 text-xs font-medium text-emerald-300">OpenAI</span>
                  <span class="rounded bg-blue-500/20 px-2 py-0.5 text-xs font-medium text-blue-300">{{ t('home.pricing.tags.efficient') }}</span>
                </div>
                <h3 class="text-xl font-bold text-white">GPT-5.2 Codex</h3>
                <p class="mt-1 text-sm text-gray-400">{{ t('home.pricing.models.gpt.desc') }}</p>
              </div>
              <div class="p-6">
                <div class="mb-4 grid grid-cols-2 gap-4">
                  <div>
                    <div class="text-xs text-gray-500">{{ t('home.pricing.input') }}</div>
                    <div class="text-2xl font-bold text-white">$1.75<span class="text-sm font-normal text-gray-500"> / MTok</span></div>
                  </div>
                  <div>
                    <div class="text-xs text-gray-500">{{ t('home.pricing.output') }}</div>
                    <div class="text-2xl font-bold text-white">$14<span class="text-sm font-normal text-gray-500"> / MTok</span></div>
                  </div>
                </div>
                <div class="rounded-lg bg-white/5 p-3">
                  <div class="mb-2 text-xs font-medium text-gray-400">{{ t('home.pricing.cachePrice') }}</div>
                  <div class="flex justify-between text-sm">
                    <span class="text-gray-400">{{ t('home.pricing.cacheWrite') }}</span>
                    <span class="font-semibold text-gray-500">-</span>
                  </div>
                  <div class="flex justify-between text-sm">
                    <span class="text-gray-400">{{ t('home.pricing.cacheRead') }}</span>
                    <span class="font-semibold text-white">$0.175</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <p class="mt-8 text-center text-xs text-gray-500">{{ t('home.pricing.note') }}</p>

          <!-- CTA -->
          <div class="mt-10 text-center">
            <router-link
              :to="isAuthenticated ? dashboardPath : '/login'"
              class="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-cyan-500 to-blue-600 px-8 py-3 text-base font-semibold text-white shadow-lg shadow-cyan-500/30 transition-all hover:scale-105 hover:shadow-cyan-500/50"
            >
              {{ t('home.pricing.cta') }}
              <Icon name="arrowRight" size="md" />
            </router-link>
          </div>
        </div>
      </section>

      <!-- Dark Mode Footer -->
      <footer class="border-t border-white/5 px-6 py-8 backdrop-blur-sm">
        <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 sm:flex-row">
          <p class="text-sm text-gray-400">&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
          <div v-if="docUrl" class="flex items-center gap-6">
            <a :href="docUrl" target="_blank" class="text-sm text-gray-400 transition-colors hover:text-cyan-400">{{ t('home.docs') }}</a>
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

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '星算code')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteInitial = computed(() => siteName.value.charAt(0).toUpperCase())
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(false)
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})
const currentYear = computed(() => new Date().getFullYear())

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('home-theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  const savedTheme = localStorage.getItem('home-theme')
  isDark.value = savedTheme === 'dark'
  document.documentElement.classList.toggle('dark', isDark.value)
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.hero-heading {
  background: linear-gradient(to bottom, #ffffff, #94a3b8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.aurora-gradient {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse 80% 50% at 50% -20%, rgba(0, 255, 255, 0.15), transparent),
    radial-gradient(ellipse 60% 50% at 80% 50%, rgba(255, 0, 255, 0.1), transparent),
    radial-gradient(ellipse 60% 50% at 20% 80%, rgba(0, 102, 255, 0.1), transparent);
  animation: aurora-shift 20s ease-in-out infinite;
  opacity: 0.6;
}

@keyframes aurora-shift {
  0%, 100% { opacity: 0.6; transform: scale(1); }
  50% { opacity: 0.8; transform: scale(1.1); }
}

.stars-layer {
  position: absolute;
  inset: 0;
  background-repeat: repeat;
  animation: twinkle linear infinite;
}

.stars-small {
  background-image: radial-gradient(2px 2px at 20% 30%, rgba(255, 255, 255, 0.8), transparent),
    radial-gradient(2px 2px at 60% 70%, rgba(255, 255, 255, 0.6), transparent),
    radial-gradient(1px 1px at 50% 50%, rgba(255, 255, 255, 0.4), transparent);
  background-size: 200% 200%;
  animation-duration: 8s;
}

.stars-medium {
  background-image: radial-gradient(3px 3px at 30% 20%, rgba(0, 255, 255, 0.6), transparent),
    radial-gradient(2px 2px at 70% 80%, rgba(147, 51, 234, 0.5), transparent);
  background-size: 300% 300%;
  animation-duration: 12s;
}

.stars-large {
  background-image: radial-gradient(4px 4px at 25% 50%, rgba(0, 255, 255, 0.8), transparent),
    radial-gradient(3px 3px at 75% 25%, rgba(168, 85, 247, 0.7), transparent);
  background-size: 400% 400%;
  animation-duration: 16s;
}

@keyframes twinkle {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

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

.shooting-star.delay-1 { animation-delay: 1s; left: 30%; }
.shooting-star.delay-2 { animation-delay: 2s; left: 70%; }

@keyframes shoot {
  0% { opacity: 0; transform: translateX(-100px) translateY(0) rotate(-45deg); }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { opacity: 0; transform: translateX(300px) translateY(200px) rotate(-45deg); }
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
</style>
