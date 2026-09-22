// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import type { ClientProblem } from '#shared/utils/api/problem'
import { reportProblem } from './reportProblem'

const { useMessage } = vi.hoisted(() => ({ useMessage: vi.fn() }))

mockNuxtImport('useMessage', () => useMessage)

afterEach(() => useMessage.mockReset())

const problem = (errors: ClientProblem['errors']): ClientProblem => ({
  kind: 'problem',
  status: 422,
  code: 'VALIDATION_FAILED',
  requestId: 'req_1',
  errors
})

describe('reportProblem', () => {
  it('shows the problem and then every field error', () => {
    reportProblem(
      problem([
        { pointer: '/title', reason: 'TOO_SHORT', detail: 'too short' },
        { pointer: '/sections/0', reason: 'TOO_FEW_ITEMS', detail: 'too few' }
      ])
    )
    expect(useMessage).toHaveBeenCalledTimes(3)
    expect(useMessage.mock.calls[1]![1]).toBe('error')
    expect(useMessage.mock.calls[2]![1]).toBe('error')
    expect(String(useMessage.mock.calls[1]![0])).toMatch(/^标题：/)
    expect(String(useMessage.mock.calls[2]![0])).toMatch(/^分区：/)
  })

  it('shows one message when there are no field errors', () => {
    reportProblem(problem([]))
    expect(useMessage).toHaveBeenCalledTimes(1)
  })

  it('labels a parameter and a header error too, and falls back to the raw key', () => {
    reportProblem(
      problem([
        { parameter: 'limit', reason: 'TOO_LARGE', detail: 'too large' },
        { header: 'Idempotency-Key', reason: 'REQUIRED', detail: 'required' },
        { pointer: '/unknown_field', reason: 'REQUIRED', detail: 'required' }
      ])
    )
    expect(String(useMessage.mock.calls[1]![0])).toMatch(/^limit：/)
    expect(String(useMessage.mock.calls[2]![0])).toMatch(/^Idempotency-Key：/)
    expect(String(useMessage.mock.calls[3]![0])).toMatch(/^unknown_field：/)
  })
})
