import { settle } from '#shared/utils/api/problem'
import type { PollCreate, PollPatch } from '#shared/utils/api/schemas'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

export const usePoll = (topicId: MaybeRefOrGetter<string>) => {
  const api = useApiClient()
  const createKey = useIdempotencyKey()

  const createPoll = async (body: PollCreate) => {
    const target = toValue(topicId)
    const result = await settle(
      api.POST('/topics/{topic_id}/polls', {
        params: {
          path: { topic_id: target },
          header: {
            'Idempotency-Key': createKey.take(`/topics/${target}/polls`, body)
          }
        },
        body
      })
    )
    if (result.ok) {
      createKey.clear()
    }
    return result
  }

  const updatePoll = (pollId: string, body: PollPatch) =>
    settle(
      api.PATCH('/polls/{poll_id}', {
        params: { path: { poll_id: pollId } },
        body
      })
    )

  const deletePoll = async (pollId: string) => {
    const confirmed = await useComponentMessageStore().alert(
      '确定要删除这个投票吗？',
      '删除投票后, 所有投票数据都将丢失, 该操作不可恢复!'
    )
    if (!confirmed) {
      return undefined
    }
    return settle(
      api.DELETE('/polls/{poll_id}', { params: { path: { poll_id: pollId } } })
    )
  }

  const setVote = (pollId: string, optionIds: string[]) =>
    settle(
      api.PUT('/polls/{poll_id}/vote', {
        params: { path: { poll_id: pollId } },
        body: { option_ids: optionIds }
      })
    )

  const clearVote = (pollId: string) =>
    settle(
      api.DELETE('/polls/{poll_id}/vote', {
        params: { path: { poll_id: pollId } }
      })
    )

  return { createPoll, updatePoll, deletePoll, setVote, clearVote }
}
