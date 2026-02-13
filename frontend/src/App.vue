<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { onMounted, watch, ref } from 'vue'
import Toast from '@/components/common/Toast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import AnnouncementModal from '@/components/announcement/AnnouncementModal.vue'
import { useAppStore, useAuthStore, useSubscriptionStore } from '@/stores'
import { getSetupStatus } from '@/api/setup'
import { announcementAPI } from '@/api/announcement'
import { sanitizeUrl } from '@/utils/url'

const showAnnouncementModal = ref(false)

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()

const DEFAULT_FAVICON = '/logo.png'

/**
 * Update favicon dynamically with URL validation
 * @param logoUrl - URL of the logo to use as favicon (will be sanitized)
 */
function updateFavicon(logoUrl: string) {
  // Sanitize URL to prevent injection attacks
  const safeUrl = sanitizeUrl(logoUrl, { allowRelative: true, allowDataUrl: true })
  const finalUrl = safeUrl || DEFAULT_FAVICON

  // Remove ALL existing favicon links to avoid browser inconsistency
  // (Backend SSR may inject one, and we may have created another)
  document.querySelectorAll<HTMLLinkElement>('link[rel="icon"]').forEach((el) => el.remove())

  // Create a fresh favicon link
  const link = document.createElement('link')
  link.rel = 'icon'

  // Determine MIME type: check data URI first, then file extension
  if (finalUrl.startsWith('data:image/')) {
    // Extract MIME type from data URI (e.g., "data:image/png;base64,...")
    const mimeMatch = finalUrl.match(/^data:(image\/[^;,]+)/)
    link.type = mimeMatch ? mimeMatch[1] : 'image/png'
  } else if (finalUrl.endsWith('.svg')) {
    link.type = 'image/svg+xml'
  }
  // For other formats, don't set type - let browser auto-detect

  link.href = finalUrl
  document.head.appendChild(link)
}

// Watch for site settings changes and update favicon/title
// Note: Backend SSR already injects the correct favicon via <link rel="icon">
// This watch handles dynamic updates (e.g., admin changes logo while page is open)
watch(
  () => appStore.siteLogo,
  (newLogo, oldLogo) => {
    // Handle both value changes and clearing (fallback to default)
    if (newLogo !== oldLogo) {
      updateFavicon(newLogo || '')
    }
  },
  { immediate: false } // Don't run immediately - backend already handled initial favicon
)

// Note: Backend SSR already sets the correct title via server-side replacement
// This watch handles dynamic updates (e.g., admin changes site name while page is open)
watch(
  () => appStore.siteName,
  (newName, oldName) => {
    if (newName && newName !== oldName) {
      document.title = `${newName} - AI API Gateway`
    }
  },
  { immediate: false } // Don't run immediately - backend already handled initial title
)

// Watch for authentication state and manage subscription data
watch(
  () => authStore.isAuthenticated,
  async (isAuthenticated) => {
    if (isAuthenticated) {
      // User logged in: preload subscriptions and start polling
      subscriptionStore.fetchActiveSubscriptions().catch((error) => {
        console.error('Failed to preload subscriptions:', error)
      })
      subscriptionStore.startPolling()

      // Check for unread announcements
      try {
        const announcements = await announcementAPI.getUnreadAnnouncements()
        if (announcements.length > 0) {
          showAnnouncementModal.value = true
        }
      } catch (error) {
        console.error('Failed to check unread announcements:', error)
      }
    } else {
      // User logged out: clear data and stop polling
      subscriptionStore.clear()
    }
  },
  { immediate: true }
)

onMounted(async () => {
  // Check if setup is needed
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      router.replace('/setup')
      return
    }
  } catch {
    // If setup endpoint fails, assume normal mode and continue
  }

  // Load public settings into appStore (will be cached for other components)
  await appStore.fetchPublicSettings()

  // Fallback: If backend didn't inject favicon (non-embed mode) or injected a different one,
  // ensure favicon matches the loaded settings
  const existingFavicons = document.querySelectorAll<HTMLLinkElement>('link[rel="icon"]')
  const expectedLogo = appStore.siteLogo || ''

  // Check if current favicon matches expected (compare last one if multiple exist)
  const lastFavicon = existingFavicons[existingFavicons.length - 1]
  const currentHref = lastFavicon?.href || ''

  // Update if: no favicon, multiple favicons (cleanup), default with custom expected, or mismatch
  const needsUpdate =
    existingFavicons.length === 0 ||
    existingFavicons.length > 1 || // Clean up duplicate icon links (SSR + default)
    (currentHref.endsWith('/logo.png') && expectedLogo) ||
    (expectedLogo && !currentHref.includes(expectedLogo.slice(0, 50))) // partial match for data URIs

  if (needsUpdate) {
    updateFavicon(expectedLogo)
  }
})
</script>

<template>
  <NavigationProgress />
  <RouterView />
  <Toast />
  <AnnouncementModal
    :show="showAnnouncementModal"
    @close="showAnnouncementModal = false"
    @read="showAnnouncementModal = false"
  />
</template>
