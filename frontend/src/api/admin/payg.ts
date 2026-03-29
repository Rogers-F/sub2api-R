import { apiClient } from '../client'
import type { PaygAdminWallet } from '@/types'

export async function getWallet(): Promise<PaygAdminWallet> {
  const { data } = await apiClient.get<PaygAdminWallet>('/admin/payg/wallet')
  return data
}

export const paygAPI = {
  getWallet,
}

export default paygAPI
