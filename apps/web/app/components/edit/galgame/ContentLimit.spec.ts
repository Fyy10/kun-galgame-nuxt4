// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import ContentLimit from './ContentLimit.vue'
import { usePersistEditGalgameStore } from '~/store/modules/edit/galgame'

// The submitter is the only person who has seen the cover art, and a default
// nobody looked at put 961 works on catalog's SFW shelf with only explicit
// cover art — the election had nothing safe to elect, so every viewer got the
// blurred stand-in instead of the real cover (infra audit-cover-shelf,
// 2026-09-13). Both assertions here fail the moment a default comes back.
describe('EditGalgameContentLimit', () => {
  it('starts with neither SFW nor NSFW chosen', async () => {
    const wrapper = await mountSuspended(ContentLimit)

    const radios = wrapper.findAll('[role="radio"]')
    expect(radios).toHaveLength(2)
    expect(
      radios.filter((r) => r.attributes('aria-checked') === 'true')
    ).toHaveLength(0)
    expect(usePersistEditGalgameStore().content_limit).toBe('')
  })

  it('writes the chosen verdict to the store without persisting it', async () => {
    const wrapper = await mountSuspended(ContentLimit)
    const store = usePersistEditGalgameStore()

    await wrapper.findAll('[role="radio"]')[1]!.trigger('click')

    expect(store.content_limit).toBe('nsfw')
    expect(localStorage.getItem('KUNGalgameEditGalgame') ?? '').not.toContain(
      'content_limit'
    )
  })
})
