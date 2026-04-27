import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('group locale message syntax', () => {
  it.each([
    ['zh', zh],
    ['en', en]
  ])('does not use bare empty placeholders in group dialog messages for %s', (_locale, messages) => {
    const groupMessages = messages.admin.groups.claudeToolArgumentsRepair

    expect(groupMessages.tooltip).not.toContain('{}')
    expect(groupMessages.hint).not.toContain('{}')
  })
})
