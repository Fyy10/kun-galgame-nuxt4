import type { Ref } from 'vue'
import type { Reply } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const previewCache = new Map<string, Reply | null>()

export interface QuotePreviewState {
  visible: boolean
  top: number
  left: number
  loading: boolean
  reply: Reply | null
}

export const useQuoteContent = (containerRef: Ref<HTMLElement | null>) => {
  const api = useApiClient()
  const preview = reactive<QuotePreviewState>({
    visible: false,
    top: 0,
    left: 0,
    loading: false,
    reply: null
  })

  let hideTimer: ReturnType<typeof setTimeout> | null = null
  let fetchSeq = 0

  const clearHideTimer = () => {
    if (hideTimer) {
      clearTimeout(hideTimer)
      hideTimer = null
    }
  }

  const FLASH = [
    'outline-2',
    'outline-offset-2',
    'outline-primary',
    'rounded-lg'
  ]
  const scrollToFloor = (floor: number) => {
    const el = document.querySelector<HTMLElement>(`[id^="${floor}."]`)
    if (!el) {
      useMessage('该楼层可能在其他分页，暂时无法跳转', 'info')
      return
    }
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    el.classList.add(...FLASH)
    setTimeout(() => el.classList.remove(...FLASH), 1500)
  }

  const showPreview = async (el: HTMLElement, replyId: string) => {
    clearHideTimer()
    const rect = el.getBoundingClientRect()
    preview.top = rect.bottom + 8
    preview.left = rect.left
    preview.visible = true

    if (previewCache.has(replyId)) {
      preview.reply = previewCache.get(replyId) ?? null
      preview.loading = false
      return
    }

    preview.reply = null
    preview.loading = true
    const seq = ++fetchSeq
    const result = await settle(
      api.GET('/replies/{reply_id}', {
        params: { path: { reply_id: replyId } }
      })
    )
    if (seq !== fetchSeq) {
      return
    }
    const data = result.ok ? result.data : null
    if (result.ok || result.problem.status === 404) {
      previewCache.set(replyId, data)
    }
    preview.reply = data
    preview.loading = false
  }

  const hidePreview = () => {
    clearHideTimer()
    hideTimer = setTimeout(() => {
      preview.visible = false
    }, 200)
  }

  const keepPreview = () => {
    clearHideTimer()
  }

  const quoteFrom = (e: Event) =>
    (e.target as HTMLElement | null)?.closest<HTMLElement>('.kun-quote') ?? null

  const onClick = (e: MouseEvent) => {
    const quote = quoteFrom(e)
    if (!quote) {
      return
    }
    e.preventDefault()
    const floor = Number(quote.dataset.floor)
    if (floor > 0) {
      scrollToFloor(floor)
    }
  }

  const onOver = (e: MouseEvent) => {
    const quote = quoteFrom(e)
    if (!quote) {
      return
    }
    const replyId = quote.dataset.replyId
    if (replyId) {
      showPreview(quote, replyId)
    }
  }

  const onOut = (e: MouseEvent) => {
    if (quoteFrom(e)) {
      hidePreview()
    }
  }

  const setup = () => {
    const c = containerRef.value
    if (!c) {
      return
    }
    c.addEventListener('click', onClick)
    c.addEventListener('mouseover', onOver)
    c.addEventListener('mouseout', onOut)
  }

  const cleanup = () => {
    clearHideTimer()
    const c = containerRef.value
    if (!c) {
      return
    }
    c.removeEventListener('click', onClick)
    c.removeEventListener('mouseover', onOver)
    c.removeEventListener('mouseout', onOut)
  }

  watch(
    containerRef,
    (newEl, oldEl) => {
      if (oldEl) {
        cleanup()
      }
      if (newEl) {
        nextTick(setup)
      }
    },
    { flush: 'post' }
  )

  return { preview, keepPreview, hidePreview }
}
