import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockPost = vi.hoisted(() => vi.fn())

vi.mock('@/api/client', () => ({
  apiClient: {
    post: mockPost,
  },
}))

describe('keysAPI', () => {
  beforeEach(() => {
    vi.resetModules()
    mockPost.mockReset()
  })

  it('posts selected key ids and target group id for batch group update', async () => {
    mockPost.mockResolvedValue({ data: { updated: 2 } })
    const { keysAPI } = await import('@/api/keys')

    await expect(keysAPI.batchUpdateGroup([1, 2], 10)).resolves.toEqual({ updated: 2 })
    expect(mockPost).toHaveBeenCalledWith('/keys/batch/group', {
      ids: [1, 2],
      group_id: 10,
    })
  })
})
