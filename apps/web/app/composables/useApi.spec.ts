// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useApi } from './useApi'

const listBody = { object: 'list', items: [] }

const jsonResponse = (status: number, body: unknown, contentType: string) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': contentType }
  })

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useApi', () => {
  it('exposes a success result', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => jsonResponse(200, listBody, 'application/json'))
    )
    const { data, problem } = await useApi('topics-ok', (api) =>
      api.GET('/topics', {})
    )
    expect(data.value).toEqual(listBody)
    expect(problem.value).toBeNull()
  })

  it('exposes a problem result with data undefined', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () =>
        jsonResponse(
          400,
          {
            type: 'about:blank',
            title: 'Invalid parameter',
            status: 400,
            code: 'INVALID_PARAMETER',
            request_id: 'req_01ARZ3NDEKTSV4RRFFQ69G5FAV',
            errors: []
          },
          'application/problem+json'
        )
      )
    )
    const { data, problem } = await useApi('topics-problem', (api) =>
      api.GET('/topics', {})
    )
    expect(data.value).toBeUndefined()
    expect(problem.value).toMatchObject({
      kind: 'problem',
      status: 400,
      code: 'INVALID_PARAMETER'
    })
  })

  it('refetches when a reactive key changes', async () => {
    const fetchSpy = vi.fn(async () =>
      jsonResponse(200, listBody, 'application/json')
    )
    vi.stubGlobal('fetch', fetchSpy)
    const key = ref('page-a')
    await useApi(key, (api) => api.GET('/topics', {}))
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    key.value = 'page-b'
    await vi.waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledTimes(2)
    })
  })
})
