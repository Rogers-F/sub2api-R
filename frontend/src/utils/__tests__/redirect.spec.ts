import { describe, expect, it } from 'vitest'

import { sanitizeRedirectPath } from '@/utils/redirect'

describe('sanitizeRedirectPath', () => {
  it('returns a safe internal absolute path unchanged', () => {
    expect(sanitizeRedirectPath('/chat')).toBe('/chat')
    expect(sanitizeRedirectPath('/admin/dashboard')).toBe('/admin/dashboard')
    expect(sanitizeRedirectPath('/usage?tab=keys')).toBe('/usage?tab=keys')
  })

  it('falls back when the value is missing or empty', () => {
    expect(sanitizeRedirectPath(undefined)).toBe('/chat')
    expect(sanitizeRedirectPath(null)).toBe('/chat')
    expect(sanitizeRedirectPath('')).toBe('/chat')
  })

  it('rejects external / protocol-relative / scheme targets', () => {
    expect(sanitizeRedirectPath('//evil.com')).toBe('/chat')
    expect(sanitizeRedirectPath('https://evil.com')).toBe('/chat')
    expect(sanitizeRedirectPath('/x://y')).toBe('/chat')
    expect(sanitizeRedirectPath('relative/path')).toBe('/chat')
    expect(sanitizeRedirectPath('/\\evil.com')).toBe('/chat')
    expect(sanitizeRedirectPath('/a\nb')).toBe('/chat')
  })

  it('falls back without throwing when given a duplicated query param (array)', () => {
    // Vue Router yields an array for ?redirect=/chat&redirect=//evil.com
    expect(sanitizeRedirectPath(['/chat', '//evil.com'])).toBe('/chat')
    expect(sanitizeRedirectPath([])).toBe('/chat')
    expect(sanitizeRedirectPath({ not: 'a string' })).toBe('/chat')
  })

  it('honors a custom fallback', () => {
    expect(sanitizeRedirectPath(undefined, '/dashboard')).toBe('/dashboard')
  })
})
