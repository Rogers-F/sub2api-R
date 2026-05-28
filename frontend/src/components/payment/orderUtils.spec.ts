import { afterEach, describe, expect, it, vi } from 'vitest'
import { formatDateTimeInTimezone } from '@/utils/format'
import { formatOrderDateTime } from './orderUtils'

vi.mock('@/i18n', () => ({
  getLocale: () => 'en-US',
  i18n: {
    global: {
      t: (key: string) => key
    }
  }
}))

describe('payment orderUtils', () => {
  const originalTZ = process.env.TZ

  afterEach(() => {
    process.env.TZ = originalTZ
  })

  it('formats order timestamps in Beijing time instead of browser local time', () => {
    process.env.TZ = 'UTC'
    const value = '2026-01-01T16:30:00Z'

    expect(formatOrderDateTime(value)).toBe(formatDateTimeInTimezone(value, 'Asia/Shanghai'))
  })
})
