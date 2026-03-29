import { describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const { getReferralInfoMock, getReferralSettingsMock, getReferralRewardsMock } = vi.hoisted(() => ({
  getReferralInfoMock: vi.fn(),
  getReferralSettingsMock: vi.fn(),
  getReferralRewardsMock: vi.fn()
}))

vi.stubGlobal('localStorage', {
  getItem: vi.fn(() => null),
  setItem: vi.fn(),
  removeItem: vi.fn()
})

vi.mock('@/i18n', () => ({
  getLocale: () => 'zh-CN'
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn()
  })
}))

vi.mock('@/api/referral', () => ({
  referralAPI: {
    getReferralInfo: getReferralInfoMock,
    getReferralSettings: getReferralSettingsMock,
    getReferralRewards: getReferralRewardsMock
  }
}))

vi.mock('@/utils/clipboard', () => ({
  copyToClipboard: vi.fn()
}))

vi.mock('qrcode', () => ({
  default: {
    toDataURL: vi.fn()
  }
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

import ReferralView from '../ReferralView.vue'

describe('ReferralView', () => {
  it('renders a custom 0% commission rate instead of falling back to the global rate', async () => {
    getReferralInfoMock.mockResolvedValue({
      enabled: true,
      referral_code: 'abc123',
      commission_rate: 0,
      total_invited: 0,
      total_reward: 0,
      register_reward: 0,
      commission_reward: 0
    })
    getReferralSettingsMock.mockResolvedValue({
      enabled: true,
      register_bonus: 5,
      commission_rate: 0.3
    })
    getReferralRewardsMock.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 10,
      pages: 1
    })

    const wrapper = mount(ReferralView, {
      global: {
        stubs: {
          AppLayout: {
            template: '<div><slot /></div>'
          },
          Icon: true,
          Teleport: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('0%')
    expect(wrapper.text()).not.toContain('30%')
  })
})
