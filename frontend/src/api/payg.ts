import { apiClient } from './client'
import type { PaygOrder, PaygPrecreateResult, PaygWallet } from '@/types'

export interface PaygPrecreateRequest {
  amount: number
  payway: string
}

export async function getWallet(): Promise<PaygWallet> {
  const { data } = await apiClient.get<PaygWallet>('/user/payg/wallet')
  return data
}

export async function precreate(payload: PaygPrecreateRequest): Promise<PaygPrecreateResult> {
  const { data } = await apiClient.post<PaygPrecreateResult>('/user/payg/precreate', payload)
  return data
}

export async function queryOrder(orderID: number): Promise<PaygOrder> {
  const { data } = await apiClient.post<PaygOrder>(`/user/payg/orders/${orderID}/query`)
  return data
}

export const paygAPI = {
  getWallet,
  precreate,
  queryOrder,
}

export default paygAPI
