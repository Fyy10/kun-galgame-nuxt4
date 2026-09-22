// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { useIdempotencyKey } from './useIdempotencyKey'

describe('useIdempotencyKey', () => {
  it('reuses the key for the same payload, renews it when the payload changes, and clears after success', () => {
    const keys = useIdempotencyKey()
    const first = { title: 'a' }
    const a = keys.take(first)
    const b = keys.take({ title: 'a' })
    expect(b).toBe(a)
    const c = keys.take({ title: 'b' })
    expect(c).not.toBe(a)
    keys.clear()
    const d = keys.take({ title: 'b' })
    expect(d).not.toBe(c)
  })
})
