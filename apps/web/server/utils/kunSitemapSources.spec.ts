import { describe, expect, it, vi } from 'vitest'
import { createApiClient } from '#shared/utils/api/client'
import { collectTopicUrls } from './kunSitemapSources'

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' }
  })

const topicItem = (id: string, bumpedAt: string) => ({
  object: 'topic',
  id,
  title: id,
  state: 'published',
  category: 'galgame',
  sections: [],
  cover_images: [],
  user: { object: 'user', id: '1', name: 'n', avatar: null },
  view_count: 0,
  like_count: 0,
  reply_count: 0,
  comment_count: 0,
  has_best_answer: false,
  mini_apps: [],
  is_nsfw: false,
  bumped_at: bumpedAt,
  created_at: '2020-01-01T00:00:00Z',
  upvoted_at: null
})

const listBody = (ids: Array<[string, string]>, next?: string) => ({
  object: 'list',
  items: ids.map(([id, bumped]) => topicItem(id, bumped)),
  ...(next ? { next_cursor: next } : {})
})

describe('collectTopicUrls', () => {
  it('walks three pages and emits every id once, in order, with lastmod equal to bumped_at', async () => {
    const pages = [
      listBody(
        [
          ['1', '2026-01-01T00:00:00.000Z'],
          ['2', '2026-01-02T00:00:00.000Z']
        ],
        'c1'
      ),
      listBody(
        [
          ['3', '2026-01-03T00:00:00.000Z'],
          ['4', '2026-01-04T00:00:00.000Z']
        ],
        'c2'
      ),
      listBody([['5', '2026-01-05T00:00:00.000Z']])
    ]
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      expect(url.searchParams.get('limit')).toBe('100')
      expect(url.searchParams.get('include_nsfw')).toBeNull()
      expect(input.headers.get('cookie')).toBeNull()
      const cursor = url.searchParams.get('cursor')
      if (!cursor) {
        return jsonResponse(200, pages[0])
      }
      if (cursor === 'c1') {
        return jsonResponse(200, pages[1])
      }
      expect(cursor).toBe('c2')
      return jsonResponse(200, pages[2])
    })
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const urls = await collectTopicUrls(api)
    expect(urls.map((url) => url.loc)).toEqual([
      '/topic/1',
      '/topic/2',
      '/topic/3',
      '/topic/4',
      '/topic/5'
    ])
    expect(urls.map((url) => url.lastmod)).toEqual([
      '2026-01-01T00:00:00.000Z',
      '2026-01-02T00:00:00.000Z',
      '2026-01-03T00:00:00.000Z',
      '2026-01-04T00:00:00.000Z',
      '2026-01-05T00:00:00.000Z'
    ])
    expect(urls.every((url) => url.changefreq === 'daily')).toBe(true)
    expect(urls.every((url) => url.priority === 0.8)).toBe(true)
    expect(fetchSpy).toHaveBeenCalledTimes(3)
  })

  it('keeps page one URLs when page two fails', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const cursor = new URL(input.url).searchParams.get('cursor')
      if (!cursor) {
        return jsonResponse(
          200,
          listBody([['1', '2026-01-01T00:00:00.000Z']], 'c1')
        )
      }
      return jsonResponse(500, { error: 'nope' })
    })
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const urls = await collectTopicUrls(api)
    expect(urls).toEqual([
      {
        loc: '/topic/1',
        lastmod: '2026-01-01T00:00:00.000Z',
        changefreq: 'daily',
        priority: 0.8
      }
    ])
    expect(fetchSpy).toHaveBeenCalledTimes(2)
  })

  it('stops a cursor that never ends after 200 pages', async () => {
    const fetchSpy = vi.fn(async () =>
      jsonResponse(
        200,
        listBody([['1', '2026-01-01T00:00:00.000Z']], 'forever')
      )
    )
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const urls = await collectTopicUrls(api)
    expect(fetchSpy).toHaveBeenCalledTimes(200)
    expect(urls).toHaveLength(200)
  })
})
