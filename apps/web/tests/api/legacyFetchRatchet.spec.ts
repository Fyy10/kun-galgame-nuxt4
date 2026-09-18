// @vitest-environment node
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  writeFileSync
} from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const CALL = /\b(?:useKunFetch|kunFetch)\s*[<(]/g

type WalkResult = { count: number; files: number }

export const countLegacyFetchCalls = (roots: string[]): WalkResult => {
  const files: string[] = []
  const walk = (dir: string) => {
    let entries
    try {
      entries = readdirSync(dir, { withFileTypes: true })
    } catch {
      return
    }
    for (const entry of entries) {
      const path = join(dir, entry.name)
      if (entry.isDirectory()) {
        walk(path)
        continue
      }
      if (
        entry.isFile() &&
        (entry.name.endsWith('.ts') || entry.name.endsWith('.vue')) &&
        !entry.name.endsWith('.spec.ts')
      ) {
        files.push(path)
      }
    }
  }
  for (const root of roots) {
    walk(root)
  }
  let count = 0
  for (const file of files) {
    const text = readFileSync(file, 'utf8')
    const matches = text.match(CALL)
    if (matches) {
      count += matches.length
    }
  }
  return { count, files: files.length }
}

export const legacyFetchRatchetViolations = (
  roots: string[],
  baseline: number
): string[] => {
  const { count, files } = countLegacyFetchCalls(roots)
  if (files === 0) {
    return [
      'scanned zero .ts/.vue files; the tree must contain application sources'
    ]
  }
  if (count > baseline) {
    return [
      `legacy fetch count ${count} exceeds baseline ${baseline}; remove kunFetch/useKunFetch calls`
    ]
  }
  if (count < baseline) {
    return [
      `legacy fetch count ${count} is below baseline ${baseline}; set tests/api/legacy-fetch-baseline to ${count}`
    ]
  }
  return []
}

const webRoot = fileURLToPath(new URL('../..', import.meta.url))
const baseline = Number(
  readFileSync(join(webRoot, 'tests/api/legacy-fetch-baseline'), 'utf8').trim()
)

describe('F4 legacy fetch ratchet', () => {
  it('matches the committed baseline', () => {
    expect(
      legacyFetchRatchetViolations(
        [
          join(webRoot, 'app'),
          join(webRoot, 'server'),
          join(webRoot, 'shared')
        ],
        baseline
      )
    ).toEqual([])
  })

  it('counts calls across ts and vue, ignoring spec files', () => {
    const root = mkdtempSync(join(tmpdir(), 'legacy-fetch-'))
    writeFileSync(
      join(root, 'a.ts'),
      'kunFetch("/x")\nuseKunFetch<Foo>("/y")\n'
    )
    writeFileSync(join(root, 'b.vue'), '<script>kunFetch("/z")</script>\n')
    writeFileSync(
      join(root, 'c.spec.ts'),
      'kunFetch("/nope")\nuseKunFetch("/nope")\n'
    )
    expect(countLegacyFetchCalls([root])).toEqual({ count: 3, files: 2 })
  })

  it('fails when the tree has no files', () => {
    const root = mkdtempSync(join(tmpdir(), 'legacy-fetch-empty-'))
    mkdirSync(join(root, 'empty'))
    expect(legacyFetchRatchetViolations([join(root, 'empty')], 0)).toEqual([
      'scanned zero .ts/.vue files; the tree must contain application sources'
    ])
  })
})
