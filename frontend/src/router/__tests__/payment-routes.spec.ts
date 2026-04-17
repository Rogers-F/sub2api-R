import { describe, expect, it, vi } from 'vitest'

vi.stubGlobal('localStorage', {
  getItem: vi.fn(() => null),
  setItem: vi.fn(),
  removeItem: vi.fn(),
})

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    isAdmin: false,
    isSimpleMode: false,
    checkAuth: vi.fn(),
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    siteName: 'Test Site',
    backendModeEnabled: false,
    cachedPublicSettings: {
      payment_enabled: true,
      custom_menu_items: [],
    },
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

describe('payment routes', () => {
  it('注册用户侧 payment 相关路由', async () => {
    const { default: router } = await import('@/router')

    const orderList = router.getRoutes().find((route) => route.name === 'OrderList')
    const paymentQr = router.getRoutes().find((route) => route.name === 'PaymentQRCode')
    const paymentResult = router.getRoutes().find((route) => route.name === 'PaymentResult')
    const stripePayment = router.getRoutes().find((route) => route.name === 'StripePayment')
    const stripePopup = router.getRoutes().find((route) => route.name === 'StripePopup')

    expect(orderList?.path).toBe('/orders')
    expect(orderList?.meta.requiresPayment).toBe(true)

    expect(paymentQr?.path).toBe('/payment/qrcode')
    expect(paymentQr?.meta.requiresPayment).toBe(true)

    expect(paymentResult?.path).toBe('/payment/result')
    expect(paymentResult?.meta.requiresAuth).toBe(false)

    expect(stripePayment?.path).toBe('/payment/stripe')
    expect(stripePayment?.meta.requiresPayment).toBe(true)

    expect(stripePopup?.path).toBe('/payment/stripe-popup')
    expect(stripePopup?.meta.requiresPayment).toBe(true)
  })
})
