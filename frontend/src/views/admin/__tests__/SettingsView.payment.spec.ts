import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'

import SettingsView from '../SettingsView.vue'

const {
  getSettings,
  getAdminApiKey,
  getOverloadCooldownSettings,
  getStreamTimeoutSettings,
  getRectifierSettings,
  getBetaPolicySettings,
  getWebSearchEmulationConfig,
  getAllGroups,
  getPaymentProviders,
  listProxies,
} = vi.hoisted(() => ({
  localStorageStub: vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  }),
  getSettings: vi.fn(),
  getAdminApiKey: vi.fn(),
  getOverloadCooldownSettings: vi.fn(),
  getStreamTimeoutSettings: vi.fn(),
  getRectifierSettings: vi.fn(),
  getBetaPolicySettings: vi.fn(),
  getWebSearchEmulationConfig: vi.fn(),
  getAllGroups: vi.fn(),
  getPaymentProviders: vi.fn(),
  listProxies: vi.fn(),
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings,
      getAdminApiKey,
      getOverloadCooldownSettings,
      getStreamTimeoutSettings,
      getRectifierSettings,
      getBetaPolicySettings,
      getWebSearchEmulationConfig,
      updateWebSearchEmulationConfig: vi.fn(),
      testWebSearchEmulation: vi.fn(),
      resetWebSearchUsage: vi.fn(),
    },
    groups: {
      getAll: getAllGroups,
    },
    proxies: {
      list: listProxies,
    },
    payment: {
      getProviders: getPaymentProviders,
      createProvider: vi.fn(),
      updateProvider: vi.fn(),
      deleteProvider: vi.fn(),
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    fetchPublicSettings: vi.fn(),
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    fetch: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}))

vi.mock('@/utils/registrationEmailPolicy', () => ({
  isRegistrationEmailSuffixDomainValid: vi.fn(() => true),
  normalizeRegistrationEmailSuffixDomain: vi.fn((value: string) => value),
  normalizeRegistrationEmailSuffixDomains: vi.fn((value: string[]) => value),
  parseRegistrationEmailSuffixWhitelistInput: vi.fn(() => []),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

function createSettingsResponse() {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    frontend_url: '',
    invitation_code_enabled: false,
    totp_enabled: false,
    totp_encryption_key_configured: false,
    default_balance: 0,
    default_concurrency: 1,
    default_subscriptions: [],
    site_name: 'Sub2API',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    payment_enabled: true,
    payment_min_amount: 1,
    payment_max_amount: 1000,
    payment_daily_limit: 5000,
    payment_order_timeout_minutes: 30,
    payment_max_pending_orders: 3,
    payment_enabled_types: ['alipay'],
    payment_balance_disabled: false,
    payment_balance_recharge_multiplier: 1,
    payment_recharge_fee_rate: 0,
    payment_load_balance_strategy: 'round-robin',
    payment_product_name_prefix: '',
    payment_product_name_suffix: '',
    payment_help_image_url: '',
    payment_help_text: '',
    payment_cancel_rate_limit_enabled: false,
    payment_cancel_rate_limit_max: 10,
    payment_cancel_rate_limit_window: 1,
    payment_cancel_rate_limit_unit: 'day',
    payment_cancel_rate_limit_window_mode: 'rolling',
    purchase_subscription_enabled: false,
    purchase_subscription_url: '',
    payg_enabled: false,
    payg_exchange_rate: 1,
    payg_fixed_amount_options: [],
    shouqianba_terminal_sn: '',
    shouqianba_terminal_key_configured: false,
    backend_mode_enabled: false,
    custom_menu_items: [],
    custom_endpoints: [],
    smtp_host: '',
    smtp_port: 587,
    smtp_username: '',
    smtp_password_configured: false,
    smtp_from_email: '',
    smtp_from_name: '',
    smtp_use_tls: true,
    turnstile_enabled: false,
    turnstile_site_key: '',
    turnstile_secret_key_configured: false,
    linuxdo_connect_enabled: false,
    linuxdo_connect_client_id: '',
    linuxdo_connect_client_secret_configured: false,
    linuxdo_connect_redirect_url: '',
    enable_model_fallback: false,
    fallback_model_anthropic: '',
    fallback_model_openai: '',
    fallback_model_gemini: '',
    fallback_model_antigravity: '',
    enable_identity_patch: false,
    identity_patch_prompt: '',
    ops_monitoring_enabled: false,
    ops_realtime_monitoring_enabled: false,
    ops_query_mode_default: 'auto',
    ops_metrics_interval_seconds: 60,
    min_claude_code_version: '',
    max_claude_code_version: '',
    allow_ungrouped_key_scheduling: false,
    balance_low_notify_enabled: false,
    balance_low_notify_threshold: 0,
    balance_low_notify_recharge_url: '',
    account_quota_notify_enabled: false,
    account_quota_notify_emails: [],
  }
}

describe('admin SettingsView payment tab', () => {
  beforeEach(() => {
    getSettings.mockReset()
    getAdminApiKey.mockReset()
    getOverloadCooldownSettings.mockReset()
    getStreamTimeoutSettings.mockReset()
    getRectifierSettings.mockReset()
    getBetaPolicySettings.mockReset()
    getWebSearchEmulationConfig.mockReset()
    getAllGroups.mockReset()
    getPaymentProviders.mockReset()
    listProxies.mockReset()

    getSettings.mockResolvedValue(createSettingsResponse())
    getAdminApiKey.mockResolvedValue({ exists: false, masked_key: '' })
    getOverloadCooldownSettings.mockResolvedValue({ enabled: true, cooldown_minutes: 10 })
    getStreamTimeoutSettings.mockResolvedValue({
      enabled: false,
      action: 'temp_unsched',
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10,
    })
    getRectifierSettings.mockResolvedValue({
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true,
      apikey_signature_enabled: false,
      apikey_signature_patterns: [],
    })
    getBetaPolicySettings.mockResolvedValue({ rules: [] })
    getWebSearchEmulationConfig.mockResolvedValue({ enabled: false, providers: [] })
    getAllGroups.mockResolvedValue([])
    getPaymentProviders.mockResolvedValue({ data: [] })
    listProxies.mockResolvedValue({ items: [] })
  })

  it('显示 payment 页签并在启用时渲染 provider 管理区域', async () => {
    const wrapper = shallowMount(SettingsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          Select: true,
          GroupBadge: true,
          GroupOptionItem: true,
          Toggle: true,
          ImageUpload: true,
          BackupSettings: true,
          PaymentProviderList: { template: '<div data-test="payment-provider-list" />' },
          PaymentProviderDialog: true,
          ConfirmDialog: true,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()

    const paymentTab = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.settings.tabs.payment')
    )
    expect(paymentTab).toBeTruthy()

    await paymentTab!.trigger('click')
    await flushPromises()

    expect(wrapper.html()).toContain('payment-provider-list')
  })

  it('在 gateway 页签中渲染 web search 模拟配置区', async () => {
    const wrapper = shallowMount(SettingsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          Select: true,
          GroupBadge: true,
          GroupOptionItem: true,
          Toggle: true,
          ImageUpload: true,
          BackupSettings: true,
          PaymentProviderList: true,
          PaymentProviderDialog: true,
          ConfirmDialog: true,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()

    const gatewayTab = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.settings.tabs.gateway')
    )
    expect(gatewayTab).toBeTruthy()

    await gatewayTab!.trigger('click')
    await flushPromises()

    expect(wrapper.html()).toContain('admin.settings.webSearchEmulation.title')
  })
})
