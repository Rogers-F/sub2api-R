import type { Announcement, CreateAnnouncementRequest, UpdateAnnouncementRequest, BasePaginationResponse } from '@/types'
import { apiClient } from './client'

// User-facing announcement API
export const announcementAPI = {
  // Get unread announcements for current user
  getUnreadAnnouncements(): Promise<Announcement[]> {
    return apiClient.get('/announcements/unread')
  },

  // Mark a single announcement as read
  markAsRead(id: number): Promise<void> {
    return apiClient.post(`/announcements/${id}/read`)
  },

  // Mark multiple announcements as read
  markAllAsRead(announcementIds: number[]): Promise<void> {
    return apiClient.post('/announcements/read-all', {
      announcement_ids: announcementIds
    })
  }
}

// Admin announcement API
export const adminAnnouncementAPI = {
  // List all announcements
  list(page = 1, pageSize = 20): Promise<BasePaginationResponse<Announcement>> {
    return apiClient.get('/admin/announcements', {
      params: { page, page_size: pageSize }
    })
  },

  // Get announcement by ID
  get(id: number): Promise<Announcement> {
    return apiClient.get(`/admin/announcements/${id}`)
  },

  // Create announcement
  create(data: CreateAnnouncementRequest): Promise<Announcement> {
    return apiClient.post('/admin/announcements', data)
  },

  // Update announcement
  update(id: number, data: UpdateAnnouncementRequest): Promise<Announcement> {
    return apiClient.put(`/admin/announcements/${id}`, data)
  },

  // Delete announcement
  delete(id: number): Promise<void> {
    return apiClient.delete(`/admin/announcements/${id}`)
  }
}
