// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import FavoriteToggle from './Toggle.vue'

const { kunFetch } = vi.hoisted(() => ({
  kunFetch: vi.fn()
}))

mockNuxtImport('kunFetch', () => kunFetch)

afterEach(() => {
  kunFetch.mockReset()
  usePersistUserStore().resetUser()
})

describe('FavoriteToggle', () => {
  it('calls kunFetch with the endpoint and body when action is absent', async () => {
    kunFetch.mockResolvedValue('ok')
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(FavoriteToggle, {
      props: {
        favorited: false,
        count: 3,
        endpoint: '/galgame-quiz/9/favorite',
        body: { galgame_quiz_id: 9 }
      }
    })
    const reaction = wrapper.findComponent({ name: 'KunReaction' })
    await reaction.vm.$emit('change', true)
    expect(kunFetch).toHaveBeenCalledWith('/galgame-quiz/9/favorite', {
      method: 'PUT',
      body: { galgame_quiz_id: 9 }
    })
  })

  it('calls action instead of kunFetch when action is present', async () => {
    const action = vi.fn(async () => true)
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(FavoriteToggle, {
      props: {
        favorited: true,
        count: 4,
        action
      }
    })
    const reaction = wrapper.findComponent({ name: 'KunReaction' })
    await reaction.vm.$emit('change', false)
    expect(action).toHaveBeenCalledWith(false)
    expect(kunFetch).not.toHaveBeenCalled()
  })
})
