// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { useIdempotencyKey } from './useIdempotencyKey'

describe('useIdempotencyKey', () => {
  it('reuses the key for the same request, renews it when the payload changes, and clears after success', () => {
    const keys = useIdempotencyKey()
    const a = keys.take('/topics', { title: 'a' })
    const b = keys.take('/topics', { title: 'a' })
    expect(b).toBe(a)
    const c = keys.take('/topics', { title: 'b' })
    expect(c).not.toBe(a)
    keys.clear()
    const d = keys.take('/topics', { title: 'b' })
    expect(d).not.toBe(c)
  })

  // The modal is mounted once in app.vue, so the key outlives the route. The
  // server fingerprints the path too: reusing it across topics answered
  // 409 IDEMPOTENCY_KEY_REUSED for 24 hours after any failed upvote.
  it('renews the key when the same payload goes to a different target', () => {
    const keys = useIdempotencyKey()
    const a = keys.take('/topics/1/upvotes', { note: 'same' })
    const b = keys.take('/topics/2/upvotes', { note: 'same' })
    expect(b).not.toBe(a)
    const again = keys.take('/topics/2/upvotes', { note: 'same' })
    expect(again).toBe(b)
  })
})
