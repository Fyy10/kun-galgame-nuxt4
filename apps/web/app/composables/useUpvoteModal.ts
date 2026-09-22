import type { TopicUpvote } from '#shared/utils/api/schemas'

export interface UpvoteTarget {
  topicId: string
  targetUserId: number
}

const isOpen = ref(false)
const target = ref<UpvoteTarget | null>(null)
const lastCreated = ref<TopicUpvote | null>(null)
let settleUpvote: ((pushed: TopicUpvote | false) => void) | null = null

export const useUpvoteModal = () => {
  const open = (t: UpvoteTarget) =>
    new Promise<TopicUpvote | false>((resolve) => {
      target.value = t
      settleUpvote = resolve
      isOpen.value = true
    })

  const close = (result: TopicUpvote | false) => {
    if (result) {
      lastCreated.value = result
    }
    const resolve = settleUpvote
    settleUpvote = null
    isOpen.value = false
    resolve?.(result)
  }

  return { isOpen, target, lastCreated, open, close }
}
