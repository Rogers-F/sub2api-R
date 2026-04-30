import { apiClient } from '../client'
import type { Account, Enterprise, PaginatedResponse } from '@/types'

export interface EnterpriseListFilters {
  search?: string
  status?: string
}

export interface EnterprisePayload {
  name: string
  notes?: string | null
  status?: 'active' | 'disabled'
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: EnterpriseListFilters
): Promise<PaginatedResponse<Enterprise>> {
  const { data } = await apiClient.get<PaginatedResponse<Enterprise>>('/admin/enterprises', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    }
  })
  return data
}

export async function listActive(): Promise<Enterprise[]> {
  const result = await list(1, 500, { status: 'active' })
  return result.items || []
}

export async function getById(id: number): Promise<Enterprise> {
  const { data } = await apiClient.get<Enterprise>(`/admin/enterprises/${id}`)
  return data
}

export async function create(payload: EnterprisePayload): Promise<Enterprise> {
  const { data } = await apiClient.post<Enterprise>('/admin/enterprises', payload)
  return data
}

export async function update(id: number, payload: EnterprisePayload): Promise<Enterprise> {
  const { data } = await apiClient.put<Enterprise>(`/admin/enterprises/${id}`, payload)
  return data
}

export async function deleteEnterprise(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/enterprises/${id}`)
  return data
}

export async function listAccounts(
  id: number,
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    platform?: string
    type?: string
    status?: string
    group?: string
    search?: string
    privacy_mode?: string
  }
): Promise<PaginatedResponse<Account>> {
  const { data } = await apiClient.get<PaginatedResponse<Account>>(`/admin/enterprises/${id}/accounts`, {
    params: {
      page,
      page_size: pageSize,
      ...filters
    }
  })
  return data
}

export async function assignAccounts(id: number, accountIds: number[]): Promise<{ moved: number }> {
  const { data } = await apiClient.post<{ moved: number }>(`/admin/enterprises/${id}/accounts`, {
    account_ids: accountIds
  })
  return data
}

export async function unassignAccounts(id: number, accountIds: number[]): Promise<{ moved: number }> {
  const { data } = await apiClient.delete<{ moved: number }>(`/admin/enterprises/${id}/accounts`, {
    data: {
      account_ids: accountIds
    }
  })
  return data
}

export const enterprisesAPI = {
  list,
  listActive,
  getById,
  create,
  update,
  delete: deleteEnterprise,
  listAccounts,
  assignAccounts,
  unassignAccounts
}

export default enterprisesAPI
