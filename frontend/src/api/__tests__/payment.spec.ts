import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('../client', () => ({
  apiClient: {
    get: mocks.get,
    post: mocks.post,
    put: mocks.put,
    delete: mocks.delete
  }
}))

describe('payment API exports', () => {
  beforeEach(() => {
    mocks.get.mockReset()
    mocks.post.mockReset()
    mocks.put.mockReset()
    mocks.delete.mockReset()
  })

  it('exposes user payment endpoints via api index', async () => {
    mocks.get.mockResolvedValue({ data: {} })
    mocks.post.mockResolvedValue({ data: {} })

    const api = await import('../index')

    await api.paymentAPI.getConfig()
    await api.paymentAPI.getCheckoutInfo()
    await api.paymentAPI.createOrder({
      amount: 100,
      payment_type: 'alipay',
      order_type: 'balance'
    })

    expect(mocks.get).toHaveBeenNthCalledWith(1, '/payment/config')
    expect(mocks.get).toHaveBeenNthCalledWith(2, '/payment/checkout-info')
    expect(mocks.post).toHaveBeenCalledWith('/payment/orders', {
      amount: 100,
      payment_type: 'alipay',
      order_type: 'balance'
    })
  })

  it('exposes admin payment endpoints via admin API barrel', async () => {
    mocks.get.mockResolvedValue({ data: {} })
    mocks.put.mockResolvedValue({ data: {} })

    const { adminAPI } = await import('../admin')

    await adminAPI.payment.getConfig()
    await adminAPI.payment.getOrders({ page: 2, status: 'PENDING' })
    await adminAPI.payment.updateConfig({ enabled: true, min_amount: 10 })

    expect(mocks.get).toHaveBeenNthCalledWith(1, '/admin/payment/config')
    expect(mocks.get).toHaveBeenNthCalledWith(2, '/admin/payment/orders', {
      params: { page: 2, status: 'PENDING' }
    })
    expect(mocks.put).toHaveBeenCalledWith('/admin/payment/config', {
      enabled: true,
      min_amount: 10
    })
  })
})
