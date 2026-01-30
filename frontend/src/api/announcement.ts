import type { Announcement, CreateAnnouncementRequest, UpdateAnnouncementRequest, BasePaginationResponse } from '@/types'
import { apiClient } from './client'

// User-facing announcement API
export const announcementAPI = {
  // Get unread announcements for current user
  async getUnreadAnnouncements(): Promise<Announcement[]> {
    const { data } = await apiClient.get('/announcements/unread')
    return data
  },

  // Mark a single announcement as read
  async markAsRead(id: number): Promise<void> {
    await apiClient.post(`/announcements/${id}/read`)
  },

  // Mark multiple announcements as read
  async markAllAsRead(announcementIds: number[]): Promise<void> {
    await apiClient.post('/announcements/read-all', {
      announcement_ids: announcementIds
    })
  }
}

// Admin announcement API
export const adminAnnouncementAPI = {
  // List all announcements
  async list(page = 1, pageSize = 20): Promise<BasePaginationResponse<Announcement>> {
    const { data } = await apiClient.get('/admin/announcements', {
      params: { page, page_size: pageSize }
    })
    return data
  },

  // Get announcement by ID
  async get(id: number): Promise<Announcement> {
    const { data } = await apiClient.get(`/admin/announcements/${id}`)
    return data
  },

  // Create announcement
  async create(payload: CreateAnnouncementRequest): Promise<Announcement> {
    const { data } = await apiClient.post('/admin/announcements', payload)
    return data
  },

  // Update announcement
  async update(id: number, payload: UpdateAnnouncementRequest): Promise<Announcement> {
    const { data } = await apiClient.put(`/admin/announcements/${id}`, payload)
    return data
  },

  // Delete announcement
  async delete(id: number): Promise<void> {
    await apiClient.delete(`/admin/announcements/${id}`)
  }
}
