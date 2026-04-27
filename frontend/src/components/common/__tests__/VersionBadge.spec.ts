import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import VersionBadge from '../VersionBadge.vue'

const appStore = reactive({
  versionLoading: false,
  currentVersion: '0.2.120',
  latestVersion: '',
  hasUpdate: false,
  buildType: 'release',
  versionWarning: 'GitHub API returned 404',
  fetchVersion: vi.fn(),
  clearVersionCache: vi.fn()
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    isAdmin: true
  }),
  useAppStore: () => appStore
}))

vi.mock('@/api/admin/system', () => ({
  performUpdate: vi.fn(),
  restartService: vi.fn()
}))

describe('VersionBadge', () => {
  beforeEach(() => {
    appStore.versionLoading = false
    appStore.currentVersion = '0.2.120'
    appStore.latestVersion = ''
    appStore.hasUpdate = false
    appStore.buildType = 'release'
    appStore.versionWarning = 'GitHub API returned 404'
    appStore.fetchVersion.mockReset()
    appStore.clearVersionCache.mockReset()
  })

  it('shows update check warnings instead of claiming the current version is latest', async () => {
    const wrapper = mount(VersionBadge, {
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.find('button').trigger('click')

    expect(wrapper.text()).toContain('version.checkFailed')
    expect(wrapper.text()).toContain('GitHub API returned 404')
    expect(wrapper.text()).not.toContain('version.upToDate')
  })
})
