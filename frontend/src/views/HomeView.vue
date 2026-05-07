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

  <div v-else class="min-h-screen">
    <!-- ==================== Top announcement bar ==================== -->
    <div class="border-b hairline bg-paper-100/80 dark:bg-ink-900/60">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-2.5 text-xs text-dust-500 dark:text-pearl-300">
        <span class="flex items-center gap-2">
          <span class="xs-pulse-dot"></span>
          {{ t('home.announcement.statusOk') }}
        </span>
        <span class="hidden md:flex items-center gap-4">
          <a v-if="docUrl" :href="docUrl" target="_blank" class="hover:text-dust-700 dark:hover:text-pearl-100 transition-colors">{{ t('home.docs') }}</a>
          <span class="text-dust-300 dark:text-pearl-500">·</span>
          <span>v{{ appVersion }}</span>
        </span>
      </div>
    </div>

    <!-- ==================== Nav ==================== -->
    <header class="sticky top-0 z-50 glass border-b">
      <nav class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <a class="flex items-center gap-3 group" href="#top">
          <div class="relative h-9 w-9 rounded-xl border hairline bg-gradient-to-b from-paper-200 to-paper-300 dark:from-ink-700 dark:to-ink-900 flex items-center justify-center overflow-hidden">
            <img v-if="siteLogo" :src="siteLogo" :alt="siteName" class="h-full w-full object-contain" />
            <svg v-else viewBox="0 0 32 32" class="h-5 w-5">
              <defs>
                <linearGradient id="hv-logo-grad" x1="0" x2="1" y1="0" y2="1">
                  <stop offset="0" stop-color="#F1E4C8"/>
                  <stop offset="1" stop-color="#A88347"/>
                </linearGradient>
              </defs>
              <g fill="none" stroke="url(#hv-logo-grad)" stroke-width="1.4" stroke-linecap="round">
                <circle cx="9" cy="9" r="1.4" fill="url(#hv-logo-grad)"/>
                <circle cx="23" cy="11" r="1.6" fill="url(#hv-logo-grad)"/>
                <circle cx="16" cy="20" r="1.8" fill="url(#hv-logo-grad)"/>
                <circle cx="6" cy="22" r="1.2" fill="url(#hv-logo-grad)"/>
                <circle cx="26" cy="24" r="1.2" fill="url(#hv-logo-grad)"/>
                <path d="M9 9 L23 11 L16 20 L6 22 M16 20 L26 24"/>
              </g>
            </svg>
          </div>
          <div class="leading-tight">
            <div class="font-display text-xl tracking-wide text-dust-900 dark:text-pearl-50">{{ siteName }}</div>
            <div class="text-[10px] uppercase tracking-[0.22em] text-dust-400 dark:text-pearl-400">{{ t('home.tagline') }}</div>
          </div>
        </a>

        <div class="hidden md:flex items-center gap-8 text-sm text-dust-700 dark:text-pearl-100">
          <a class="hover:text-dust-900 dark:hover:text-white transition-colors" href="#models">{{ t('home.nav.models') }}</a>
          <a class="hover:text-dust-900 dark:hover:text-white transition-colors" href="#pricing">{{ t('home.nav.pricing') }}</a>
          <a class="hover:text-dust-900 dark:hover:text-white transition-colors" href="#why">{{ t('home.nav.why') }}</a>
          <a v-if="docUrl" :href="docUrl" target="_blank" class="hover:text-dust-900 dark:hover:text-white transition-colors">{{ t('home.docs') }}</a>
        </div>

        <div class="flex items-center gap-3">
          <LocaleSwitcher />
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-dust-500 hover:text-dust-900 hover:bg-paper-100 dark:text-pearl-300 dark:hover:text-white dark:hover:bg-white/[0.06] transition-colors"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" />
          </button>
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="hidden md:inline-flex text-sm text-dust-700 hover:text-dust-900 dark:text-pearl-200 dark:hover:text-white px-3 py-2 transition-colors"
          >
            {{ t('home.dashboard') }}
          </router-link>
          <router-link
            v-else
            to="/login"
            class="hidden md:inline-flex text-sm text-dust-700 hover:text-dust-900 dark:text-pearl-200 dark:hover:text-white px-3 py-2 transition-colors"
          >
            {{ t('home.login') }}
          </router-link>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="btn-gold inline-flex items-center gap-2 rounded-full px-5 py-2.5 text-sm font-medium transition-all"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
            <Icon name="arrowRight" size="md" />
          </router-link>
        </div>
      </nav>
    </header>

    <!-- ==================== HERO ==================== -->
    <section id="top" class="relative overflow-hidden">
      <!-- 暗色专属氛围层 -->
      <div class="absolute inset-0 pointer-events-none overflow-hidden hidden dark:block">
        <div class="absolute left-1/2 -top-[20%] h-[1100px] w-[1100px] -translate-x-1/2 rounded-full" style="background: radial-gradient(closest-side, rgba(212,182,129,0.12), transparent 70%);"></div>
        <div class="absolute left-[20%] -bottom-[30%] h-[800px] w-[800px] rounded-full" style="background: radial-gradient(closest-side, rgba(123,150,232,0.10), transparent 70%);"></div>
      </div>
      <div class="absolute inset-0 stars-bg pointer-events-none animate-drift opacity-80 hidden dark:block"></div>
      <div class="absolute inset-0 grid-pattern opacity-70 pointer-events-none"></div>

      <div class="relative mx-auto max-w-7xl px-6 py-24 md:py-32 text-center">
        <div class="inline-flex items-center gap-2 rounded-full border hairline bg-paper-0/60 dark:bg-white/[0.03] px-4 py-1.5 text-xs text-dust-500 dark:text-pearl-200">
          <span class="text-gold-600 dark:text-gold-300 font-mono tracking-wider">v{{ appVersion }}</span>
          <span class="h-3 w-px bg-dust-300/40 dark:bg-white/10"></span>
          <span>{{ t('home.hero.badge') }}</span>
        </div>

        <h1 class="font-display mx-auto mt-8 max-w-4xl text-5xl md:text-7xl leading-[1.05] tracking-tight">
          <span class="text-dust-800 dark:pearl-text">{{ t('home.hero.titleLead') }}</span><br/>
          <span class="italic">{{ t('home.hero.titleMid') }} <span class="gold-text">{{ t('home.hero.titleAccent') }}</span> {{ t('home.hero.titleTail') }}</span>
        </h1>

        <p class="mx-auto mt-7 max-w-2xl text-base md:text-lg text-dust-600 dark:text-pearl-200 leading-relaxed">
          {{ t('home.hero.subtitle') }}<br/>
          {{ t('home.hero.description') }}
        </p>

        <div class="mt-10 flex flex-wrap items-center justify-center gap-3">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="btn-gold inline-flex items-center gap-2 rounded-full px-7 py-3.5 text-sm font-medium"
          >
            {{ isAuthenticated ? t('home.hero.ctaGoToDashboard') : t('home.hero.ctaPrimary') }}
            <Icon name="arrowRight" size="md" />
          </router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            class="inline-flex items-center gap-2 rounded-full border hairline bg-paper-0 dark:bg-white/[0.04] px-7 py-3.5 text-sm text-dust-700 dark:text-pearl-100 hover:bg-paper-100 dark:hover:bg-white/[0.08] transition-colors"
          >
            <Icon name="book" size="sm" />
            {{ t('home.hero.ctaSecondary') }}
          </a>
        </div>

        <!-- Console preview -->
        <div class="relative mx-auto mt-20 max-w-5xl">
          <div class="absolute -inset-px rounded-2xl bg-gradient-to-br from-gold-300/20 via-transparent to-aurora-500/20 blur-xl"></div>
          <div class="relative rounded-2xl glass-strong p-1 ring-1 ring-paper-200 dark:ring-white/10 shadow-2xl">
            <div class="flex items-center justify-between px-4 py-2.5 border-b hairline">
              <div class="flex items-center gap-2">
                <span class="h-2.5 w-2.5 rounded-full bg-[#FF5F57]"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-[#FEBC2E]"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-[#28C840]"></span>
                <span class="ml-3 text-xs text-dust-500 dark:text-pearl-300 font-mono">~/projects/{{ siteSlug }}-quickstart</span>
              </div>
              <div class="text-xs text-dust-400 dark:text-pearl-400 font-mono">node · v22</div>
            </div>
            <div class="px-6 py-7 text-left font-mono text-[13.5px] leading-7">
              <div><span class="text-dust-400 dark:text-pearl-400">import</span> <span class="text-aurora-600 dark:text-aurora-400">Anthropic</span> <span class="text-dust-400 dark:text-pearl-400">from</span> <span class="text-mint-600 dark:text-mint-500">"@anthropic-ai/sdk"</span></div>
              <div class="mt-2"><span class="text-dust-400 dark:text-pearl-400">const</span> client <span class="text-dust-400 dark:text-pearl-400">=</span> <span class="text-dust-400 dark:text-pearl-400">new</span> <span class="text-aurora-600 dark:text-aurora-400">Anthropic</span>({</div>
              <div class="pl-6">baseURL: <span class="text-mint-600 dark:text-mint-500">"{{ apiBaseUrl }}"</span>,</div>
              <div class="pl-6">apiKey:&nbsp;&nbsp;<span class="text-mint-600 dark:text-mint-500">"sk-•••••••••••••••••••"</span></div>
              <div>})</div>
              <div class="mt-3"><span class="text-dust-400 dark:text-pearl-400">const</span> r <span class="text-dust-400 dark:text-pearl-400">=</span> <span class="text-dust-400 dark:text-pearl-400">await</span> client.messages.<span class="text-gold-600 dark:text-gold-300">create</span>({</div>
              <div class="pl-6">model: <span class="text-mint-600 dark:text-mint-500">"claude-opus-4-6"</span>,</div>
              <div class="pl-6">max_tokens: <span class="text-aurora-600 dark:text-aurora-400">1024</span>,</div>
              <div class="pl-6">messages: [{ role: <span class="text-mint-600 dark:text-mint-500">"user"</span>, content: <span class="text-mint-600 dark:text-mint-500">"hello"</span> }]</div>
              <div>})</div>
              <div class="mt-4 flex items-center gap-2 text-dust-500 dark:text-pearl-300">
                <span class="text-mint-600 dark:text-mint-500">✓</span> 200 OK
                <span class="text-dust-300 dark:text-pearl-500">·</span>
                <span class="text-dust-400 dark:text-pearl-400">cache</span>
                <span class="text-mint-600 dark:text-mint-500">hit</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ==================== Models / Pricing ==================== -->
    <section id="pricing" class="relative">
      <div class="mx-auto max-w-7xl px-6 py-24">
        <div class="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-16">
          <div>
            <div class="text-[11px] uppercase tracking-[0.3em] text-gold-600 dark:text-gold-300">{{ t('home.pricing.eyebrow') }}</div>
            <h2 class="font-display text-4xl md:text-5xl mt-3 leading-tight">{{ t('home.pricing.titleLead') }}<span class="italic gold-text">{{ t('home.pricing.titleAccent') }}</span></h2>
            <p class="mt-3 text-dust-600 dark:text-pearl-200 max-w-xl">{{ t('home.pricing.subtitle') }}</p>
          </div>
        </div>

        <div id="models" class="grid md:grid-cols-2 lg:grid-cols-4 gap-5">
          <!-- Opus 4.6 -->
          <div class="relative rounded-2xl border hairline bg-paper-0 dark:bg-ink-800/60 p-6 hover:border-dust-300 dark:hover:border-white/15 transition-colors">
            <div class="flex items-center justify-between">
              <div>
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">Anthropic</div>
                <div class="font-display text-xl mt-1">Claude Opus 4.6</div>
              </div>
              <span class="pill bg-paper-100 dark:bg-white/[0.06] text-dust-600 dark:text-pearl-200 border hairline">Flagship</span>
            </div>
            <p class="mt-3 text-xs text-dust-500 dark:text-pearl-300 leading-relaxed">{{ t('home.pricing.models.opus.desc') }}</p>
            <div class="mt-5 grid grid-cols-2 gap-2">
              <div class="rounded-lg bg-paper-50 dark:bg-white/[0.03] border hairline p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">{{ t('home.pricing.input') }}</div>
                <div class="mt-1 font-display text-xl">$5<span class="text-[11px] text-dust-400 dark:text-pearl-300 ml-0.5 font-sans">/M</span></div>
              </div>
              <div class="rounded-lg bg-paper-50 dark:bg-white/[0.03] border hairline p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">{{ t('home.pricing.output') }}</div>
                <div class="mt-1 font-display text-xl">$25<span class="text-[11px] text-dust-400 dark:text-pearl-300 ml-0.5 font-sans">/M</span></div>
              </div>
            </div>
            <div class="mt-2.5 flex items-center justify-between text-[11px] text-dust-400 dark:text-pearl-400 px-0.5">
              <span>{{ t('home.pricing.cacheWrite') }} $6.25</span>
              <span>{{ t('home.pricing.cacheRead') }} <span class="text-mint-600 dark:text-mint-500">$0.50</span></span>
            </div>
            <div class="mt-2 text-[10px] text-dust-300 dark:text-pearl-500 px-0.5">{{ t('home.pricing.models.opus.note') }}</div>
          </div>

          <!-- Sonnet 4.6 (featured) -->
          <div class="relative rounded-2xl border bg-gradient-to-b from-gold-300/10 to-transparent p-6 ring-gold">
            <div class="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 rounded-full text-[10px] uppercase tracking-widest btn-gold whitespace-nowrap">{{ t('home.pricing.bestValue') }}</div>
            <div class="flex items-center justify-between">
              <div>
                <div class="text-[10px] uppercase tracking-widest text-gold-600 dark:text-gold-300">Anthropic</div>
                <div class="font-display text-xl mt-1">Claude Sonnet 4.6</div>
              </div>
              <span class="pill bg-gold-300/15 text-gold-700 dark:text-gold-200 border border-gold-400/40 dark:border-gold-300/30">Daily</span>
            </div>
            <p class="mt-3 text-xs text-dust-600 dark:text-pearl-200 leading-relaxed">{{ t('home.pricing.models.sonnet.desc') }}</p>
            <div class="mt-5 grid grid-cols-2 gap-2">
              <div class="rounded-lg bg-paper-50 dark:bg-ink-900/60 border border-gold-400/30 dark:border-gold-300/20 p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-500 dark:text-pearl-300">{{ t('home.pricing.input') }}</div>
                <div class="mt-1 font-display text-xl gold-text">$3<span class="text-[11px] text-dust-500 dark:text-pearl-200 ml-0.5 font-sans">/M</span></div>
              </div>
              <div class="rounded-lg bg-paper-50 dark:bg-ink-900/60 border border-gold-400/30 dark:border-gold-300/20 p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-500 dark:text-pearl-300">{{ t('home.pricing.output') }}</div>
                <div class="mt-1 font-display text-xl gold-text">$15<span class="text-[11px] text-dust-500 dark:text-pearl-200 ml-0.5 font-sans">/M</span></div>
              </div>
            </div>
            <div class="mt-2.5 flex items-center justify-between text-[11px] text-dust-500 dark:text-pearl-300 px-0.5">
              <span>{{ t('home.pricing.cacheWrite') }} $3.75</span>
              <span>{{ t('home.pricing.cacheRead') }} <span class="text-mint-600 dark:text-mint-500">$0.30</span></span>
            </div>
            <div class="mt-2 text-[10px] text-dust-400 dark:text-pearl-400 px-0.5">{{ t('home.pricing.models.sonnet.note') }}</div>
          </div>

          <!-- Haiku 4.5 -->
          <div class="relative rounded-2xl border hairline bg-paper-0 dark:bg-ink-800/60 p-6 hover:border-dust-300 dark:hover:border-white/15 transition-colors">
            <div class="flex items-center justify-between">
              <div>
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">Anthropic</div>
                <div class="font-display text-xl mt-1">Claude Haiku 4.5</div>
              </div>
              <span class="pill bg-mint-500/10 text-mint-600 dark:text-mint-400 border border-mint-500/20">Fast</span>
            </div>
            <p class="mt-3 text-xs text-dust-500 dark:text-pearl-300 leading-relaxed">{{ t('home.pricing.models.haiku.desc') }}</p>
            <div class="mt-5 grid grid-cols-2 gap-2">
              <div class="rounded-lg bg-paper-50 dark:bg-white/[0.03] border hairline p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">{{ t('home.pricing.input') }}</div>
                <div class="mt-1 font-display text-xl">$1<span class="text-[11px] text-dust-400 dark:text-pearl-300 ml-0.5 font-sans">/M</span></div>
              </div>
              <div class="rounded-lg bg-paper-50 dark:bg-white/[0.03] border hairline p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">{{ t('home.pricing.output') }}</div>
                <div class="mt-1 font-display text-xl">$5<span class="text-[11px] text-dust-400 dark:text-pearl-300 ml-0.5 font-sans">/M</span></div>
              </div>
            </div>
            <div class="mt-2.5 flex items-center justify-between text-[11px] text-dust-400 dark:text-pearl-400 px-0.5">
              <span>{{ t('home.pricing.cacheWrite') }} $1.25</span>
              <span>{{ t('home.pricing.cacheRead') }} <span class="text-mint-600 dark:text-mint-500">$0.10</span></span>
            </div>
          </div>

          <!-- GPT-5.2 Codex -->
          <div class="relative rounded-2xl border hairline bg-paper-0 dark:bg-ink-800/60 p-6 hover:border-dust-300 dark:hover:border-white/15 transition-colors">
            <div class="flex items-center justify-between">
              <div>
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">OpenAI</div>
                <div class="font-display text-xl mt-1">GPT-5.2 Codex</div>
              </div>
              <span class="pill bg-aurora-500/10 text-aurora-600 dark:text-aurora-400 border border-aurora-500/20">Coding</span>
            </div>
            <p class="mt-3 text-xs text-dust-500 dark:text-pearl-300 leading-relaxed">{{ t('home.pricing.models.gpt.desc') }}</p>
            <div class="mt-5 grid grid-cols-2 gap-2">
              <div class="rounded-lg bg-paper-50 dark:bg-white/[0.03] border hairline p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">{{ t('home.pricing.input') }}</div>
                <div class="mt-1 font-display text-xl">$1.75<span class="text-[11px] text-dust-400 dark:text-pearl-300 ml-0.5 font-sans">/M</span></div>
              </div>
              <div class="rounded-lg bg-paper-50 dark:bg-white/[0.03] border hairline p-3">
                <div class="text-[10px] uppercase tracking-widest text-dust-400 dark:text-pearl-400">{{ t('home.pricing.output') }}</div>
                <div class="mt-1 font-display text-xl">$14<span class="text-[11px] text-dust-400 dark:text-pearl-300 ml-0.5 font-sans">/M</span></div>
              </div>
            </div>
            <div class="mt-2.5 flex items-center justify-between text-[11px] text-dust-400 dark:text-pearl-400 px-0.5">
              <span>{{ t('home.pricing.cacheWrite') }} —</span>
              <span>{{ t('home.pricing.cacheRead') }} <span class="text-mint-600 dark:text-mint-500">$0.175</span></span>
            </div>
          </div>
        </div>

        <p class="mt-8 text-center text-xs text-dust-400 dark:text-pearl-400">{{ t('home.pricing.note') }}</p>
      </div>
    </section>

    <!-- ==================== Why ==================== -->
    <section id="why" class="relative border-t hairline bg-paper-100 dark:bg-ink-950">
      <div class="mx-auto max-w-7xl px-6 py-24">
        <div class="text-center mb-16">
          <div class="text-[11px] uppercase tracking-[0.3em] text-gold-600 dark:text-gold-300">{{ t('home.why.eyebrow') }}</div>
          <h2 class="font-display text-4xl md:text-5xl mt-3">{{ t('home.why.titleLead') }}<span class="italic">{{ t('home.why.titleAccent') }}</span></h2>
        </div>

        <div class="grid md:grid-cols-2 lg:grid-cols-4 gap-px bg-paper-200 dark:bg-white/[0.05] rounded-2xl overflow-hidden">
          <div v-for="(feat, i) in whyFeatures" :key="i" class="bg-paper-0 dark:bg-ink-900 p-8">
            <div class="text-3xl gold-text" style="font-family: 'Playfair Display', serif; font-style: italic; font-weight: 500;">{{ String(i + 1).padStart(2, '0') }}</div>
            <h3 class="mt-4 font-display text-xl">{{ feat.title }}</h3>
            <p class="mt-3 text-sm text-dust-500 dark:text-pearl-300 leading-relaxed">{{ feat.desc }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- ==================== CTA ==================== -->
    <section class="relative border-t hairline">
      <div class="absolute inset-0 pointer-events-none overflow-hidden hidden dark:block opacity-60">
        <div class="absolute left-1/2 -top-[30%] h-[800px] w-[800px] -translate-x-1/2 rounded-full" style="background: radial-gradient(closest-side, rgba(212,182,129,0.15), transparent 70%);"></div>
      </div>
      <div class="relative mx-auto max-w-4xl px-6 py-24 text-center">
        <h2 class="font-display text-4xl md:text-6xl leading-[1.05]">
          <span class="text-dust-800 dark:pearl-text">{{ t('home.cta2.titleLead') }}</span><br/>
          <span class="italic gold-text">{{ t('home.cta2.titleAccent') }}</span>
        </h2>
        <p class="mt-6 text-dust-600 dark:text-pearl-200 max-w-xl mx-auto">{{ t('home.cta2.subtitle') }}</p>
        <div class="mt-10 flex flex-wrap items-center justify-center gap-3">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="btn-gold inline-flex items-center gap-2 rounded-full px-7 py-3.5 text-sm font-medium"
          >
            {{ isAuthenticated ? t('home.cta2.ctaGoToDashboard') : t('home.cta2.ctaPrimary') }}
            <Icon name="arrowRight" size="md" />
          </router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            class="inline-flex items-center gap-2 rounded-full border hairline bg-paper-0 dark:bg-white/[0.04] px-7 py-3.5 text-sm text-dust-700 dark:text-pearl-100 hover:bg-paper-100 dark:hover:bg-white/[0.08] transition-colors"
          >
            {{ t('home.cta2.ctaSecondary') }}
          </a>
        </div>
      </div>
    </section>

    <!-- ==================== Footer ==================== -->
    <footer class="border-t hairline bg-paper-100 dark:bg-ink-950">
      <div class="mx-auto max-w-7xl px-6 py-10 flex flex-col md:flex-row items-center justify-between gap-3 text-xs text-dust-400 dark:text-pearl-400">
        <div>© {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</div>
        <div class="flex items-center gap-4">
          <span class="inline-flex items-center gap-1.5"><span class="xs-pulse-dot"></span><span class="text-dust-500 dark:text-pearl-300">{{ t('home.footer.systemsOk') }}</span></span>
          <a v-if="docUrl" :href="docUrl" target="_blank" class="hover:text-dust-700 dark:hover:text-white transition-colors">{{ t('home.docs') }}</a>
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

const siteSlug = computed(() => {
  const n = siteName.value.toLowerCase()
  return /^[a-z0-9-]+$/.test(n) ? n : 'xingsuan'
})

const apiBaseUrl = computed(() => {
  if (typeof window === 'undefined') return 'https://api.example.com/v1'
  return `${window.location.origin}/v1`
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

// "Why Xingsuan" 4 features (使用 i18n 列表)
const whyFeatures = computed(() => [
  { title: t('home.why.features.protocol.title'), desc: t('home.why.features.protocol.desc') },
  { title: t('home.why.features.routing.title'), desc: t('home.why.features.routing.desc') },
  { title: t('home.why.features.audit.title'), desc: t('home.why.features.audit.desc') },
  { title: t('home.why.features.support.title'), desc: t('home.why.features.support.desc') }
])

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
