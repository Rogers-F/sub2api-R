import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

import en from '../locales/en'
import zh from '../locales/zh'

const accountColumnCallsites = [
  '../../views/admin/AccountsView.vue',
  '../../views/admin/EnterpriseDetailView.vue',
  '../../views/admin/ProxiesView.vue'
]

const referencedAccountColumnKeys = () =>
  [
    ...new Set(
      accountColumnCallsites.flatMap((path) => {
        const source = readFileSync(new URL(path, import.meta.url), 'utf8')
        return [...source.matchAll(/admin\.accounts\.columns\.([A-Za-z0-9_]+)/g)].map((match) => match[1])
      })
    )
  ].sort()

describe('account column locale keys', () => {
  it.each([
    ['zh', zh],
    ['en', en]
  ])('defines every account column key referenced by admin views for %s', (_locale, messages) => {
    const columnMessages = messages.admin.accounts.columns

    expect(Object.keys(columnMessages).sort()).toEqual(
      expect.arrayContaining(referencedAccountColumnKeys())
    )
  })
})
