import type { InjectionKey, Ref, ComputedRef } from 'vue'
import { settle, type ApiResult } from '#shared/utils/api/problem'
import type {
  ReactionToken,
  ReplyEngagement,
  TopicEngagement
} from '#shared/utils/api/schemas'
import { toKunReactions } from '~/utils/reactionSummary'

export interface ReactionsState {
  list: Ref<KunReaction[]>
  mineKeys: ComputedRef<string[]>
  toggle: (key: string) => Promise<void>
  topicId?: string
  replyId?: string
}

export const reactionsKey: InjectionKey<ReactionsState> = Symbol('reactions')

export type ReactionEngagement = TopicEngagement | ReplyEngagement

interface UseReactionsOptions {
  topicId?: number | string
  replyId?: number | string
  targetUserId: number
  reactions: KunReaction[]
  sync?: () => KunReaction[]
  showReactors?: boolean
  onEngagement?: (engagement: ReactionEngagement) => void
}

const MAX_AVATARS = 3

const asId = (value: number | string | undefined): string | undefined => {
  if (value === undefined || value === '' || value === 0 || value === '0') {
    return undefined
  }
  return String(value)
}

export const useReactions = (opts: UseReactionsOptions): ReactionsState => {
  const { id, name, avatar } = usePersistUserStore()
  const api = useApiClient()
  const topicId = asId(opts.topicId)
  const replyId = asId(opts.replyId)

  const clone = (rs: KunReaction[]): KunReaction[] =>
    rs.map((r) => ({
      ...r,
      reactors: r.reactors ? [...r.reactors] : undefined
    }))

  const list = ref<KunReaction[]>(opts.reactions ? clone(opts.reactions) : [])
  const inflight = new Set<string>()

  let userTouched = false
  if (opts.sync) {
    watch(opts.sync, (v) => {
      if (!userTouched) list.value = clone(v)
    })
  }

  const mineKeys = computed(() =>
    list.value.filter((r) => r.mine).map((r) => r.reaction)
  )

  const write = async (
    reaction: ReactionToken,
    held: boolean
  ): Promise<ApiResult<ReactionEngagement>> => {
    if (topicId) {
      const params = { path: { topic_id: topicId, reaction } }
      return settle(
        held
          ? api.DELETE('/topics/{topic_id}/reactions/{reaction}', { params })
          : api.PUT('/topics/{topic_id}/reactions/{reaction}', { params })
      )
    }
    const params = { path: { reply_id: replyId!, reaction } }
    return settle(
      held
        ? api.DELETE('/replies/{reply_id}/reactions/{reaction}', { params })
        : api.PUT('/replies/{reply_id}/reactions/{reaction}', { params })
    )
  }

  const removeMine = (idx: number) => {
    const r = list.value[idx]!
    r.count--
    r.mine = false
    if (r.reactors) r.reactors = r.reactors.filter((u) => u.id !== id)
    if (r.count <= 0) list.value.splice(idx, 1)
  }

  const applyOptimistic = (key: string) => {
    const idx = list.value.findIndex((r) => r.reaction === key)
    if (idx >= 0 && list.value[idx]!.mine) {
      removeMine(idx)
    } else if (idx >= 0) {
      const r = list.value[idx]!
      r.count++
      r.mine = true
      if (opts.showReactors && (r.reactors?.length ?? 0) < MAX_AVATARS) {
        ;(r.reactors ??= []).push({ id, name, avatar })
      }
    } else {
      list.value.push({
        reaction: key,
        count: 1,
        mine: true,
        reactors: opts.showReactors ? [{ id, name, avatar }] : undefined
      })
    }
    const opposite =
      key === 'like' ? 'dislike' : key === 'dislike' ? 'like' : null
    if (opposite) {
      const j = list.value.findIndex((r) => r.reaction === opposite)
      if (j >= 0 && list.value[j]!.mine) removeMine(j)
    }
  }

  const toggle = async (key: string) => {
    if (!id) {
      useAuthModal().open()
      return
    }
    if (key === 'like' && id === opts.targetUserId) {
      useMessage(10236, 'warn')
      return
    }
    if (inflight.has(key)) return
    inflight.add(key)
    userTouched = true

    const snapshot = clone(list.value)
    const held = snapshot.find((r) => r.reaction === key)?.mine === true
    applyOptimistic(key)

    const result = await write(key as ReactionToken, held)
    if (!result.ok) {
      list.value = snapshot
      reportProblem(result.problem)
      inflight.delete(key)
      return
    }
    list.value = toKunReactions(result.data.reactions)
    opts.onEngagement?.(result.data)
    inflight.delete(key)
  }

  return {
    list,
    mineKeys,
    toggle,
    topicId,
    replyId
  }
}
