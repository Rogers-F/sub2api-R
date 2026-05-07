import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'

const { createAccountMock, checkMixedChannelRiskMock, getWebSearchEmulationConfigMock } = vi.hoisted(() => ({
  createAccountMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  getWebSearchEmulationConfigMock: vi.fn()
}))

vi.stubGlobal('localStorage', {
  getItem: vi.fn(() => null),
  setItem: vi.fn(),
  removeItem: vi.fn()
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: true
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: checkMixedChannelRiskMock
    },
    settings: {
      getWebSearchEmulationConfig: getWebSearchEmulationConfigMock
    }
  }
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
}))

vi.mock('@/utils/format', () => ({
  formatDateTimeLocalInput: vi.fn(() => ''),
  parseDateTimeLocalInput: vi.fn(() => null)
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copied: { value: false },
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

function mountModal() {
  return shallowMount(CreateAccountModal, {
    props: {
      show: true,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: `
            <select
              v-bind="$attrs"
              :value="modelValue"
              @change="$emit('update:modelValue', $event.target.value)"
            >
              <option v-for="option in options" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          `
        },
        Icon: true,
        ProxySelector: true,
        GroupSelector: true,
        ModelWhitelistSelector: true,
        ConfirmDialog: true
      }
    }
  })
}

describe('CreateAccountModal', () => {
  beforeEach(() => {
    createAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    getWebSearchEmulationConfigMock.mockReset()

    createAccountMock.mockResolvedValue({})
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    getWebSearchEmulationConfigMock.mockResolvedValue({
      enabled: true,
      providers: [{ provider: 'brave' }]
    })
  })

  it('OpenAI 创建表单显示 ctx_pool WS mode 选项', async () => {
    const wrapper = mountModal()
    const openAIButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('OpenAI'))

    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')

    expect(wrapper.html()).toContain('admin.accounts.openai.wsModeCtxPool')
  })

  it('OpenAI API Key 创建时可启用 Codex preset instructions', async () => {
    const wrapper = mountModal()
    const openAIButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('OpenAI'))

    expect(openAIButton).toBeTruthy()
    await openAIButton!.trigger('click')

    const apiKeyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('API Key'))

    expect(apiKeyButton).toBeTruthy()
    await apiKeyButton!.trigger('click')
    await flushPromises()

    await wrapper.get('[data-testid="openai-codex-preset-toggle"]').trigger('click')
    await wrapper.find('input[type="text"]').setValue('OpenAI API Key')
    await wrapper.find('input[type="password"]').setValue('sk-openai-test')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.extra?.enable_codex_preset).toBe(true)
  })

  it('Anthropic API Key 创建表单显示 Web Search 覆盖选项', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const anthropicApiKeyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.claudeConsole'))

    expect(anthropicApiKeyButton).toBeTruthy()
    await anthropicApiKeyButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="web-search-emulation-mode"]').exists()).toBe(true)
  })

  it('Anthropic API Key 创建时写入 extra.web_search_emulation', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const anthropicApiKeyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.claudeConsole'))

    expect(anthropicApiKeyButton).toBeTruthy()
    await anthropicApiKeyButton!.trigger('click')
    await flushPromises()

    await wrapper.find('input[type="text"]').setValue('Anthropic API Key')
    await wrapper.find('input[type="password"]').setValue('sk-ant-test')
    await wrapper.get('[data-testid="web-search-emulation-mode"]').setValue('enabled')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.extra?.web_search_emulation).toBe('enabled')
  })
})
