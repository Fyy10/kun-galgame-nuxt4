// @vitest-environment node
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

type Registry = {
  codes: { code: string; status: number }[]
  reasons: { reason: string; param_names: string[] }[]
}

type Catalogue = {
  code: Record<string, string>
  reason: Record<string, Record<string, string>>
  status: Record<string, string>
  client: Record<string, string>
}

const PLACEHOLDER = /\{([a-z][a-z0-9_]*)\}/g
const STATUS_KEY = /^[45][0-9]{2}$/
const CLIENT_KEYS = ['network', 'timeout', 'field', 'default'] as const
const TOP_KEYS = ['code', 'reason', 'status', 'client'] as const

const placeholdersOf = (text: string): string[] | null => {
  const names: string[] = []
  const matches = text.matchAll(PLACEHOLDER)
  for (const match of matches) {
    names.push(match[1]!)
  }
  const stripped = text.replace(PLACEHOLDER, '')
  if (stripped.includes('{') || stripped.includes('}')) {
    return null
  }
  return names
}

const collectTexts = (
  node: unknown,
  path: string,
  out: { path: string; text: string }[]
) => {
  if (typeof node === 'string') {
    out.push({ path, text: node })
    return
  }
  if (node && typeof node === 'object') {
    for (const [key, value] of Object.entries(node)) {
      collectTexts(value, path ? `${path}.${key}` : key, out)
    }
  }
}

export const catalogViolations = (
  registry: Registry,
  catalogue: Catalogue
): string[] => {
  const violations: string[] = []
  const top = Object.keys(catalogue).sort()
  if (top.join(',') !== [...TOP_KEYS].sort().join(',')) {
    violations.push(`top-level keys ${top.join(',')} !== ${TOP_KEYS.join(',')}`)
  }

  const registryCodes = new Set(registry.codes.map((c) => c.code))
  const catalogueCodes = new Set(Object.keys(catalogue.code))
  for (const code of registryCodes) {
    if (!catalogueCodes.has(code)) {
      violations.push(`missing code ${code}`)
    }
  }
  for (const code of catalogueCodes) {
    if (!registryCodes.has(code)) {
      violations.push(`extra code ${code}`)
    }
  }

  const registryReasons = new Map(
    registry.reasons.map((r) => [r.reason, r.param_names])
  )
  const catalogueReasons = new Set(Object.keys(catalogue.reason))
  for (const reason of registryReasons.keys()) {
    if (!catalogueReasons.has(reason)) {
      violations.push(`missing reason ${reason}`)
    }
  }
  for (const reason of catalogueReasons) {
    if (!registryReasons.has(reason)) {
      violations.push(`extra reason ${reason}`)
    }
  }

  for (const [reason, variants] of Object.entries(catalogue.reason)) {
    const allowed = new Set(registryReasons.get(reason) ?? [])
    if (typeof variants.default !== 'string') {
      violations.push(`reason ${reason} missing default`)
    }
    for (const [key, template] of Object.entries(variants)) {
      const names = key === 'default' ? [] : key.split('__')
      if (key !== 'default') {
        if (names.length === 0 || new Set(names).size !== names.length) {
          violations.push(`reason ${reason} variant ${key} repeats params`)
        }
        const sorted = [...names].sort()
        if (sorted.join('__') !== key) {
          violations.push(`reason ${reason} variant ${key} is not ascending`)
        }
        for (const name of names) {
          if (!allowed.has(name)) {
            violations.push(
              `reason ${reason} variant ${key} uses unknown param ${name}`
            )
          }
        }
      }
      const placeholders = placeholdersOf(template)
      if (!placeholders) {
        violations.push(`reason ${reason} variant ${key} has stray braces`)
        continue
      }
      const expected = [...names].sort().join(',')
      const got = [...placeholders].sort().join(',')
      if (expected !== got) {
        violations.push(
          `reason ${reason} variant ${key} placeholders ${got} !== ${expected}`
        )
      }
    }
  }

  const noPlaceholderGroups: [string, Record<string, string>][] = [
    ['code', catalogue.code],
    ['status', catalogue.status],
    ['client', catalogue.client]
  ]
  for (const [group, texts] of noPlaceholderGroups) {
    for (const [key, text] of Object.entries(texts)) {
      const placeholders = placeholdersOf(text)
      if (!placeholders || placeholders.length > 0) {
        violations.push(`${group}.${key} has placeholders`)
      }
    }
  }

  const requiredStatuses = new Set<string>(['429', '502', '504'])
  for (const { status } of registry.codes) {
    requiredStatuses.add(String(status))
  }
  for (const key of Object.keys(catalogue.status)) {
    if (!STATUS_KEY.test(key)) {
      violations.push(`status key ${key} is not a 4xx/5xx code`)
    }
  }
  for (const status of requiredStatuses) {
    if (!(status in catalogue.status)) {
      violations.push(`missing status ${status}`)
    }
  }

  const clientKeys = Object.keys(catalogue.client).sort()
  if (clientKeys.join(',') !== [...CLIENT_KEYS].sort().join(',')) {
    violations.push(
      `client keys ${clientKeys.join(',')} !== ${CLIENT_KEYS.join(',')}`
    )
  }

  const texts: { path: string; text: string }[] = []
  collectTexts(catalogue, '', texts)
  for (const { path, text } of texts) {
    if (text.length === 0) {
      violations.push(`${path} is empty`)
    }
    if (/[@$|]/.test(text)) {
      violations.push(`${path} contains @, $ or |`)
    }
    if (placeholdersOf(text) === null) {
      violations.push(`${path} has stray braces`)
    }
  }

  return violations
}

const registry = JSON.parse(
  readFileSync(
    fileURLToPath(
      new URL('../../../api/openapi/problems.json', import.meta.url)
    ),
    'utf8'
  )
) as Registry

const catalogue = JSON.parse(
  readFileSync(
    fileURLToPath(
      new URL('../../i18n/locales/zh-CN/problem.json', import.meta.url)
    ),
    'utf8'
  )
) as Catalogue

const clone = (): Catalogue => structuredClone(catalogue)

describe('F2 problem catalogue', () => {
  it('matches the registry', () => {
    expect(catalogViolations(registry, catalogue)).toEqual([])
  })

  it('detects a code-key mismatch', () => {
    const broken = clone()
    delete broken.code.NOT_FOUND
    expect(
      catalogViolations(registry, broken).some((v) => v.includes('NOT_FOUND'))
    ).toBe(true)
  })

  it('detects a reason-key mismatch', () => {
    const broken = clone()
    delete broken.reason.REQUIRED
    expect(
      catalogViolations(registry, broken).some((v) => v.includes('REQUIRED'))
    ).toBe(true)
  })

  it('detects a reason without default', () => {
    const broken = clone()
    const { default: _dropped, ...rest } = broken.reason.TOO_LONG
    broken.reason.TOO_LONG = rest
    expect(
      catalogViolations(registry, broken).some((v) =>
        v.includes('TOO_LONG missing default')
      )
    ).toBe(true)
  })

  it('detects placeholders on a code text', () => {
    const broken = clone()
    broken.code.NOT_FOUND = 'missing {pointer}'
    expect(
      catalogViolations(registry, broken).some((v) =>
        v.includes('code.NOT_FOUND has placeholders')
      )
    ).toBe(true)
  })

  it('detects a missing required status', () => {
    const broken = clone()
    delete broken.status['429']
    expect(
      catalogViolations(registry, broken).some((v) =>
        v.includes('missing status 429')
      )
    ).toBe(true)
  })

  it('detects a client-key mismatch', () => {
    const broken = clone()
    broken.client.extra = 'nope'
    expect(
      catalogViolations(registry, broken).some((v) => v.includes('client keys'))
    ).toBe(true)
  })

  it('detects a top-level key mismatch', () => {
    const broken = clone() as Catalogue & { extra?: string }
    broken.extra = 'nope'
    expect(
      catalogViolations(registry, broken).some((v) =>
        v.includes('top-level keys')
      )
    ).toBe(true)
  })

  it('detects forbidden characters in a text', () => {
    const broken = clone()
    broken.client.default = '出错了 @admin'
    expect(
      catalogViolations(registry, broken).some((v) =>
        v.includes('contains @, $ or |')
      )
    ).toBe(true)
  })
})
