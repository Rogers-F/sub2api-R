import type { ReferralInfo, ReferralReward, ReferralSettings, BasePaginationResponse } from '@/types'
import { apiClient } from './client'

export const referralAPI = {
  // Get current user's referral info
  getReferralInfo(): Promise<ReferralInfo> {
    return apiClient.get('/user/referral')
  },

  // Get referral rewards history
  getReferralRewards(page = 1, pageSize = 20): Promise<BasePaginationResponse<ReferralReward>> {
    return apiClient.get('/user/referral/rewards', {
      params: { page, page_size: pageSize }
    })
  },

  // Get public referral settings
  getReferralSettings(): Promise<ReferralSettings> {
    return apiClient.get('/referral/settings')
  }
}
