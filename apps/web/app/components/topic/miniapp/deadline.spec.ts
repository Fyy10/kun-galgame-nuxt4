import { describe, expect, it } from 'vitest'
import { closesAtFromPicker, deadlineFromPicker } from './deadline'

describe('closesAtFromPicker', () => {
  it('drops the milliseconds /api/v1 refuses', () => {
    expect(deadlineFromPicker('2026-08-31')).toMatch(/\.\d{3}Z$/)
    expect(closesAtFromPicker('2026-08-31')).toMatch(
      /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/
    )
    expect(closesAtFromPicker('2026-08-31')).toHaveLength(20)
  })

  it('stays undefined when nothing was picked', () => {
    expect(closesAtFromPicker(undefined)).toBeUndefined()
    expect(closesAtFromPicker('')).toBeUndefined()
  })
})
