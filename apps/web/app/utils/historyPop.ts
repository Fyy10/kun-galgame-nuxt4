import type { Router } from 'vue-router'

let pendingPop = false
let currentIsPop = false

export const trackHistoryPops = (router: Router) => {
  // Back/forward is a popstate; push, replace and the first load are not.
  // Cursor lists reuse loaded pages only for a pop so scroll restoration lands
  // where the reader was.
  router.options.history.listen(() => {
    pendingPop = true
  })
  router.beforeEach(() => {
    currentIsPop = pendingPop
    pendingPop = false
  })
}

export const isHistoryPop = (): boolean => currentIsPop
