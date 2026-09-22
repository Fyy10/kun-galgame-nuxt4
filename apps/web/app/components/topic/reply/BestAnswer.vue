<script setup lang="ts">
import type { Reply, Topic } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { KUN_MOEMOEPOINT } from '~/constants/moemoepoint'
import type { ComputedRef } from 'vue'

const props = defineProps<{
  reply: Reply
}>()

const api = useApiClient()
const pageTopic = inject<ComputedRef<Topic>>('pageTopic')
const replaceTopic = inject<(topic: Topic) => void>('replaceTopic', () => {})
const canSetBestAnswer = computed(
  () => pageTopic?.value.viewer?.can_set_best_answer === true
)

const isAuthorsOwnReply = computed(
  () => pageTopic?.value.author.id === props.reply.author.id
)

const handleUpdateTopicBestAnswer = async () => {
  const points = KUN_MOEMOEPOINT.bestAnswer
  const own = isAuthorsOwnReply.value
  const res = await useComponentMessageStore().alert(
    props.reply.is_best_answer
      ? own
        ? '您确定取消设置该回复为最佳答案吗'
        : `您确定取消设置该回复为最佳答案吗, 该操作将会扣除该回复人 ${points} 萌萌点`
      : own
        ? '您确定设置这个回复为最佳答案吗，楼主自己的回复不会获得萌萌点'
        : `您确定设置这个回复为最佳答案吗， 该操作将会为该回复人增加 ${points} 萌萌点`
  )
  if (!res) {
    return
  }

  const topicId = props.reply.topic_id
  const result = await settle(
    props.reply.is_best_answer
      ? api.DELETE('/topics/{topic_id}/best-answer', {
          params: { path: { topic_id: topicId } }
        })
      : api.PUT('/topics/{topic_id}/best-answer', {
          params: { path: { topic_id: topicId } },
          body: { reply_id: props.reply.id }
        })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(
    props.reply.is_best_answer ? '取消设置最佳答案成功' : '设置最佳答案成功',
    'success'
  )
  replaceTopic(result.data)
}
</script>

<template>
  <KunButton
    v-if="canSetBestAnswer"
    variant="light"
    :color="reply.is_best_answer ? 'warning' : 'default'"
    size="sm"
    @click="handleUpdateTopicBestAnswer"
    class-name="whitespace-nowrap gap-2 justify-start"
  >
    <KunIcon class-name="text-lg" name="lucide:bookmark-check" />
    {{ reply.is_best_answer ? '取消设置最佳答案' : '将该回复设为最佳答案' }}
  </KunButton>
</template>
