import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { isHistoryPop, trackHistoryPops } from './historyPop'

const stub = { template: '<div />' }

const createTrackedRouter = () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/a', component: stub },
      { path: '/b', component: stub },
      { path: '/c', component: stub }
    ]
  })
  trackHistoryPops(router)
  return router
}

const waitForPop = (router: ReturnType<typeof createTrackedRouter>) =>
  new Promise<void>((resolve) => {
    const stop = router.afterEach(() => {
      stop()
      resolve()
    })
  })

describe('historyPop', () => {
  it('is true during back and forward, and false during push', async () => {
    const router = createTrackedRouter()
    const during: boolean[] = []
    router.beforeEach(() => {
      during.push(isHistoryPop())
    })

    await router.push('/a')
    await router.push('/b')
    expect(isHistoryPop()).toBe(false)

    const backDone = waitForPop(router)
    router.back()
    await backDone
    expect(isHistoryPop()).toBe(true)
    expect(during.at(-1)).toBe(true)

    await router.push('/c')
    expect(isHistoryPop()).toBe(false)
    expect(during.at(-1)).toBe(false)
  })

  it('is true for forward after a back', async () => {
    const router = createTrackedRouter()
    await router.push('/a')
    await router.push('/b')
    const backDone = waitForPop(router)
    router.back()
    await backDone
    const forwardDone = waitForPop(router)
    router.forward()
    await forwardDone
    expect(isHistoryPop()).toBe(true)
  })

  it('is false for replace and the initial load', async () => {
    const router = createTrackedRouter()
    await router.push('/a')
    expect(isHistoryPop()).toBe(false)
    await router.replace('/b')
    expect(isHistoryPop()).toBe(false)
  })
})
