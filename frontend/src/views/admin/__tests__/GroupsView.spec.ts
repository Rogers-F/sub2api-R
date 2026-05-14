import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent } from 'vue'
import GroupsView from '../GroupsView.vue'

const {
  groupsListMock,
  getUsageSummaryMock,
  getCapacitySummaryMock,
  getAccountByIdMock
} = vi.hoisted(() => ({
  groupsListMock: vi.fn(),
  getUsageSummaryMock: vi.fn(),
  getCapacitySummaryMock: vi.fn(),
  getAccountByIdMock: vi.fn()
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn()
  })
}))

vi.mock('@/api/admin/system', () => ({
  checkUpdates: vi.fn(),
  performUpdate: vi.fn(),
  restartService: vi.fn()
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

vi.mock('@/i18n', () => ({
  i18n: {
    global: {
      t: (key: string) => key,
      locale: { value: 'zh' },
      setLocaleMessage: vi.fn()
    }
  },
  default: {
    global: {
      t: (key: string) => key,
      locale: { value: 'zh' },
      setLocaleMessage: vi.fn()
    }
  },
  getLocale: vi.fn(() => 'zh'),
  setLocale: vi.fn(),
  loadLocaleMessages: vi.fn(),
  initI18n: vi.fn(),
  availableLocales: []
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: groupsListMock,
      getUsageSummary: getUsageSummaryMock,
      getCapacitySummary: getCapacitySummaryMock,
      getAll: vi.fn(async () => []),
      update: vi.fn(),
      create: vi.fn(),
      delete: vi.fn()
    },
    accounts: {
      list: vi.fn(async () => ({ items: [] })),
      getById: getAccountByIdMock
    }
  }
}))

const AppLayoutStub = defineComponent({
  template: '<div><slot /></div>'
})

const TablePageLayoutStub = defineComponent({
  template: '<div><slot name="filters" /><slot name="table" /><slot /></div>'
})

const DataTableStub = defineComponent({
  props: {
    data: { type: Array, default: () => [] }
  },
  template: `
    <div>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
})

const BaseDialogStub = defineComponent({
  props: {
    show: Boolean,
    title: String
  },
  emits: ['close'],
  template: `
    <div v-if="show" :data-test="title === 'admin.groups.editGroup' ? 'edit-dialog' : 'dialog'">
      <slot />
      <slot name="footer" />
    </div>
  `
})

const PassthroughStub = defineComponent({
  template: '<div><slot /></div>'
})

const SelectStub = defineComponent({
  inheritAttrs: false,
  props: ['modelValue'],
  emits: ['update:modelValue', 'change'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value); $emit(\'change\')"><slot /></select>'
})

function makeGroup() {
  return {
    id: 1,
    name: '默认分组',
    description: '',
    platform: 'anthropic',
    rate_multiplier: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard',
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    image_price_1k: null,
    image_price_2k: null,
    image_price_4k: null,
    claude_code_only: false,
    claude_prompt_caching_enabled: true,
    thinking_signature_compat_enabled: false,
    claude_tool_use_repair_enabled: false,
    claude_tool_arguments_repair_enabled: false,
    strong_safety_mode_enabled: true,
    fallback_group_id: null,
    fallback_group_id_on_invalid_request: null,
    allow_messages_dispatch: false,
    require_oauth_only: false,
    require_privacy_set: false,
    force_application_json_for_non_stream: false,
    default_mapped_model: '',
    model_routing_enabled: true,
    model_routing: {
      'claude-*': [42]
    },
    supported_model_scopes: ['claude'],
    mcp_xml_inject: true,
    account_count: 1,
    sort_order: 10,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z'
  }
}

describe('GroupsView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    groupsListMock.mockReset()
    getUsageSummaryMock.mockReset()
    getCapacitySummaryMock.mockReset()
    getAccountByIdMock.mockReset()

    groupsListMock.mockResolvedValue({
      items: [makeGroup()],
      total: 1,
      pages: 1
    })
    getUsageSummaryMock.mockResolvedValue([])
    getCapacitySummaryMock.mockResolvedValue([])
    getAccountByIdMock.mockReturnValue(new Promise(() => {}))
  })

  afterEach(() => {
    document.body.innerHTML = ''
    document.body.className = ''
  })

  const mountGroupsView = (stubs: Record<string, any> = {}) => {
    return mount(GroupsView, {
      attachTo: document.body,
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          ConfirmDialog: PassthroughStub,
          EmptyState: PassthroughStub,
          Select: SelectStub,
          PlatformIcon: PassthroughStub,
          Icon: PassthroughStub,
          GroupRateMultipliersModal: PassthroughStub,
          GroupCapacityBadge: PassthroughStub,
          Pagination: PassthroughStub,
          VueDraggable: PassthroughStub,
          ...stubs
        }
      }
    })
  }

  it('opens the create dialog when the create group button is clicked', async () => {
    const wrapper = mountGroupsView()

    await flushPromises()
    const createButton = wrapper.find('[data-tour="groups-create-btn"]')
    expect(createButton.exists()).toBe(true)
    await createButton.trigger('click')
    await flushPromises()

    expect(document.body.textContent).toContain('admin.groups.createGroup')
    expect(document.body.querySelector('#create-group-form')).not.toBeNull()
  })

  it('opens the edit dialog without waiting for model routing account lookups', async () => {
    const wrapper = mountGroupsView({ BaseDialog: BaseDialogStub })

    await flushPromises()
    const editButton = wrapper.findAll('button').find((button) => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')

    expect(getAccountByIdMock).toHaveBeenCalledWith(42)
    expect(wrapper.find('[data-test="edit-dialog"]').exists()).toBe(true)
  })

  it('renders the real edit dialog when the edit group button is clicked', async () => {
    const wrapper = mountGroupsView()

    await flushPromises()
    const editButton = wrapper.findAll('button').find((button) => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    expect(getAccountByIdMock).toHaveBeenCalledWith(42)
    expect(document.body.textContent).toContain('admin.groups.editGroup')
    expect(document.body.querySelector('#edit-group-form')).not.toBeNull()
  })
})
