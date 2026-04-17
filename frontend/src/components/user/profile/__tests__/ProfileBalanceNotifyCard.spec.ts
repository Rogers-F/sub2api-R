import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => {
  const authStore = {
    user: null as any
  }

  return {
    authStore,
    showError: vi.fn(),
    showSuccess: vi.fn(),
    updateProfile: vi.fn(),
    sendNotifyEmailCode: vi.fn(),
    verifyNotifyEmail: vi.fn(),
    removeNotifyEmail: vi.fn(),
    toggleNotifyEmail: vi.fn()
  }
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.authStore
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess
  })
}))

vi.mock('@/api', () => ({
  userAPI: {
    updateProfile: mocks.updateProfile,
    sendNotifyEmailCode: mocks.sendNotifyEmailCode,
    verifyNotifyEmail: mocks.verifyNotifyEmail,
    removeNotifyEmail: mocks.removeNotifyEmail,
    toggleNotifyEmail: mocks.toggleNotifyEmail
  }
}))

import ProfileBalanceNotifyCard from '../ProfileBalanceNotifyCard.vue'

describe('ProfileBalanceNotifyCard', () => {
  beforeEach(() => {
    mocks.authStore.user = null
    mocks.showError.mockReset()
    mocks.showSuccess.mockReset()
    mocks.updateProfile.mockReset()
    mocks.sendNotifyEmailCode.mockReset()
    mocks.verifyNotifyEmail.mockReset()
    mocks.removeNotifyEmail.mockReset()
    mocks.toggleNotifyEmail.mockReset()
  })

  it('toggles balance notifications with profile update', async () => {
    mocks.updateProfile.mockResolvedValue({ id: 1, balance_notify_enabled: false })
    const wrapper = mount(ProfileBalanceNotifyCard, {
      props: {
        enabled: true,
        threshold: 6,
        extraEmails: [],
        systemDefaultThreshold: 5,
        userEmail: 'user@example.com'
      }
    })

    const toggle = wrapper.find('input[type="checkbox"]')
    expect(toggle.exists()).toBe(true)

    await toggle.setValue(false)

    expect(mocks.updateProfile).toHaveBeenCalledWith({
      balance_notify_enabled: false
    })
  })

  it('saves custom threshold', async () => {
    mocks.updateProfile.mockResolvedValue({ id: 1, balance_notify_threshold: 9.5 })
    const wrapper = mount(ProfileBalanceNotifyCard, {
      props: {
        enabled: true,
        threshold: 6,
        extraEmails: [],
        systemDefaultThreshold: 5,
        userEmail: 'user@example.com'
      }
    })

    await wrapper.get('input[type="number"]').setValue('9.5')
    const saveButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.save'))

    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')

    expect(mocks.updateProfile).toHaveBeenCalledWith({
      balance_notify_threshold: 9.5
    })
  })
})
