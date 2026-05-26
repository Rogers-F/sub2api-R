import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'

const {
  mockList,
  mockBatchUpdateGroup,
  mockGetAvailable,
  mockGetUserGroupRates,
  mockGetDashboardApiKeysUsage,
  mockGetPublicSettings,
  mockShowSuccess,
  mockShowError,
} = vi.hoisted(() => ({
  mockList: vi.fn(),
  mockBatchUpdateGroup: vi.fn(),
  mockGetAvailable: vi.fn(),
  mockGetUserGroupRates: vi.fn(),
  mockGetDashboardApiKeysUsage: vi.fn(),
  mockGetPublicSettings: vi.fn(),
  mockShowSuccess: vi.fn(),
  mockShowError: vi.fn(),
}))

vi.mock('@/api', () => ({
  keysAPI: {
    list: mockList,
    batchUpdateGroup: mockBatchUpdateGroup,
  },
  userGroupsAPI: {
    getAvailable: mockGetAvailable,
    getUserGroupRates: mockGetUserGroupRates,
  },
  usageAPI: {
    getDashboardApiKeysUsage: mockGetDashboardApiKeysUsage,
  },
  authAPI: {
    getPublicSettings: mockGetPublicSettings,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mockShowSuccess,
    showError: mockShowError,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(() => Promise.resolve(true)),
  }),
}))

const messages: Record<string, string> = {
  'common.name': 'Name',
  'common.actions': 'Actions',
  'keys.apiKey': 'API Key',
  'keys.group': 'Group',
  'keys.usage': 'Usage',
  'keys.rateLimitColumn': 'Rate Limit',
  'keys.expiresAt': 'Expires',
  'keys.lastUsedAt': 'Last Used',
  'keys.created': 'Created',
  'keys.batchGroup.selected': '已选择 {count} 个密钥',
  'keys.batchGroup.selectCurrentPage': '本页全选',
  'keys.batchGroup.clear': '清除选择',
  'keys.batchGroup.selectTargetGroup': '选择目标分组',
  'keys.batchGroup.submit': '批量切换分组',
  'keys.batchGroup.submitting': '正在切换...',
  'keys.batchGroup.success': '已切换 {count} 个密钥的分组',
  'keys.batchGroup.failed': '批量切换分组失败',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        let value = messages[key] ?? key
        if (params) {
          for (const [paramKey, paramValue] of Object.entries(params)) {
            value = value.replace(`{${paramKey}}`, String(paramValue))
          }
        }
        return value
      },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const TablePageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>',
}

const DataTableStub = defineComponent({
  props: {
    columns: { type: Array, required: true },
    data: { type: Array, required: true },
    loading: { type: Boolean, default: false },
  },
  template: `
    <div>
      <slot name="header-select" />
      <div v-for="row in data" :key="row.id">
        <slot name="cell-select" :row="row" :value="row.select" />
        <slot name="cell-name" :row="row" :value="row.name" />
      </div>
      <slot v-if="!loading && data.length === 0" name="empty" />
    </div>
  `,
})

const SelectStub = defineComponent({
  inheritAttrs: false,
  props: {
    modelValue: { type: [String, Number, Boolean, null], default: null },
    options: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  methods: {
    emitValue(event: Event) {
      const value = (event.target as HTMLSelectElement).value
      this.$emit('update:modelValue', value === '' ? null : Number(value))
    },
  },
  template: `
    <select v-bind="$attrs" :value="modelValue ?? ''" @change="emitValue">
      <option value=""></option>
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `,
})

const makeApiKey = (id: number, name: string) => ({
  id,
  user_id: 7,
  key: `sk-test-${id}`,
  name,
  group_id: 9,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  quota: 0,
  quota_used: 0,
  expires_at: null,
  created_at: '2026-05-26T00:00:00Z',
  updated_at: '2026-05-26T00:00:00Z',
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
})

async function mountKeysView() {
  const { default: KeysView } = await import('../KeysView.vue')
  const wrapper = mount(KeysView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Select: SelectStub,
        Pagination: true,
        SearchInput: true,
        BaseDialog: true,
        ConfirmDialog: true,
        EmptyState: true,
        Icon: true,
        UseKeyModal: true,
        EndpointPopover: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Teleport: true,
      },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

describe('KeysView batch group switching', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('localStorage', {
      getItem: vi.fn(() => null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    })
    mockList.mockResolvedValue({
      items: [makeApiKey(1, 'Key One'), makeApiKey(2, 'Key Two')],
      total: 2,
      pages: 1,
    })
    mockGetAvailable.mockResolvedValue([
      {
        id: 10,
        name: 'Target Group',
        description: null,
        platform: 'anthropic',
        subscription_type: 'standard',
        rate_multiplier: 1,
      },
    ])
    mockGetUserGroupRates.mockResolvedValue({})
    mockGetDashboardApiKeysUsage.mockResolvedValue({ stats: {} })
    mockGetPublicSettings.mockResolvedValue({})
    mockBatchUpdateGroup.mockResolvedValue({ updated: 1 })
  })

  it('shows batch group bar after selecting a key', async () => {
    const wrapper = await mountKeysView()

    await wrapper.find('[data-test="key-row-select-1"]').setValue(true)

    expect(wrapper.find('[data-test="keys-batch-group-bar"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('已选择 1 个密钥')
  })

  it('submits selected ids and target group id', async () => {
    const wrapper = await mountKeysView()

    await wrapper.find('[data-test="key-row-select-1"]').setValue(true)
    await wrapper.find('[data-test="batch-group-select"]').setValue('10')
    await wrapper.find('[data-test="batch-group-submit"]').trigger('click')
    await flushPromises()

    expect(mockBatchUpdateGroup).toHaveBeenCalledWith([1], 10)
    expect(mockShowSuccess).toHaveBeenCalledWith('已切换 1 个密钥的分组')
  })
})
