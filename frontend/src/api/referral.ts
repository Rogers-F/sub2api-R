import type { ReferralInfo, ReferralReward, ReferralSettings, BasePaginationResponse } from '@/types'
import { apiClient } from './client'

export const referralAPI = {
  // Get current user's referral info
  async getReferralInfo(): Promise<ReferralInfo> {
    const { data } = await apiClient.get('/user/referral')
    return data
  },

  // Get referral rewards history
  async getReferralRewards(page = 1, pageSize = 20): Promise<BasePaginationResponse<ReferralReward>> {
    const { data } = await apiClient.get('/user/referral/rewards', {
      params: { page, page_size: pageSize }
    })
    return data
  },

  // Get public referral settings
  async getReferralSettings(): Promise<ReferralSettings> {
    const { data } = await apiClient.get('/referral/settings')
    return data
  }
}
