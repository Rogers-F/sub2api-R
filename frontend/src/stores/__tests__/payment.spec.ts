import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { usePaymentStore } from '@/stores/payment'

const mockGetConfig = vi.fn()
const mockGetPlans = vi.fn()
const mockCreateOrder = vi.fn()
const mockGetOrder = vi.fn()

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getConfig: (...args: any[]) => mockGetConfig(...args),
    getPlans: (...args: any[]) => mockGetPlans(...args),
    createOrder: (...args: any[]) => mockCreateOrder(...args),
    getOrder: (...args: any[]) => mockGetOrder(...args)
  }
}))

describe('usePaymentStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetches config once and caches it', async () => {
    mockGetConfig.mockResolvedValue({
      data: {
        payment_enabled: true,
        min_amount: 1
      }
    })

    const store = usePaymentStore()

    const first = await store.fetchConfig()
    const second = await store.fetchConfig()

    expect(first).toEqual({ payment_enabled: true, min_amount: 1 })
    expect(second).toEqual({ payment_enabled: true, min_amount: 1 })
    expect(mockGetConfig).toHaveBeenCalledTimes(1)
  })

  it('normalizes plan features and updates current order status', async () => {
    mockGetPlans.mockResolvedValue({
      data: [
        {
          id: 1,
          name: 'Monthly',
          features: 'A\nB\n'
        }
      ]
    })
    mockGetOrder.mockResolvedValue({
      data: {
        id: 99,
        status: 'PAID'
      }
    })

    const store = usePaymentStore()
    store.currentOrder = { id: 99 } as any

    const plans = await store.fetchPlans()
    const order = await store.pollOrderStatus(99)

    expect(plans[0].features).toEqual(['A', 'B'])
    expect(order).toEqual({ id: 99, status: 'PAID' })
    expect(store.currentOrder).toEqual({ id: 99, status: 'PAID' })
  })

  it('creates order through payment API', async () => {
    mockCreateOrder.mockResolvedValue({
      data: {
        order_id: 7
      }
    })

    const store = usePaymentStore()
    const result = await store.createOrder({
      amount: 88,
      payment_type: 'wxpay',
      order_type: 'balance'
    })

    expect(result).toEqual({ order_id: 7 })
    expect(mockCreateOrder).toHaveBeenCalledWith({
      amount: 88,
      payment_type: 'wxpay',
      order_type: 'balance'
    })
  })
})
