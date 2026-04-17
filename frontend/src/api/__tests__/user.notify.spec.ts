import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  delete: vi.fn()
}))

vi.mock('../client', () => ({
  apiClient: {
    get: mocks.get,
    put: mocks.put,
    post: mocks.post,
    delete: mocks.delete
  }
}))

describe('userAPI notify endpoints', () => {
  beforeEach(() => {
    mocks.get.mockReset()
    mocks.put.mockReset()
    mocks.post.mockReset()
    mocks.delete.mockReset()
  })

  it('updates balance notify fields through updateProfile', async () => {
    mocks.put.mockResolvedValue({ data: { id: 1 } })
    const { userAPI } = await import('../user')

    await userAPI.updateProfile({
      username: 'alice',
      balance_notify_enabled: true,
      balance_notify_threshold: 8.5
    })

    expect(mocks.put).toHaveBeenCalledWith('/user', {
      username: 'alice',
      balance_notify_enabled: true,
      balance_notify_threshold: 8.5
    })
  })

  it('sends verification code for notify email', async () => {
    mocks.post.mockResolvedValue({ data: {} })
    const { userAPI } = await import('../user')

    await userAPI.sendNotifyEmailCode('notify@example.com')

    expect(mocks.post).toHaveBeenCalledWith('/user/notify-email/send-code', {
      email: 'notify@example.com'
    })
  })

  it('verifies and toggles notify emails', async () => {
    mocks.post.mockResolvedValue({ data: {} })
    mocks.put.mockResolvedValue({ data: { id: 1, balance_notify_extra_emails: [] } })
    mocks.delete.mockResolvedValue({ data: {} })
    const { userAPI } = await import('../user')

    await userAPI.verifyNotifyEmail('notify@example.com', '123456')
    await userAPI.toggleNotifyEmail('notify@example.com', true)
    await userAPI.removeNotifyEmail('notify@example.com')

    expect(mocks.post).toHaveBeenCalledWith('/user/notify-email/verify', {
      email: 'notify@example.com',
      code: '123456'
    })
    expect(mocks.put).toHaveBeenCalledWith('/user/notify-email/toggle', {
      email: 'notify@example.com',
      disabled: true
    })
    expect(mocks.delete).toHaveBeenCalledWith('/user/notify-email', {
      data: { email: 'notify@example.com' }
    })
  })
})
