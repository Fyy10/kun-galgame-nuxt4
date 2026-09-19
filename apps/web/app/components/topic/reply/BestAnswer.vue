<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'

const props = defineProps<{
  reply: Reply
}>()

const { id } = usePersistUserStore()
const canSetBestAnswer = useCan('topic.set_best_answer')
const topicUserId = inject<number>('topicUserId')
const refreshTopic = inject<() => Promise<unknown>>('refreshTopic', () =>
  Promise.resolve()
)

const isDisabled = !canSetBestAnswer.value && topicUserId !== id

const handleUpdateTopicBestAnswer = async () => {
  const res = await useComponentMessageStore().alert(
    props.reply.is_best_answer
      ? '您确定取消设置该回复为最佳答案吗, 该操作将会扣除该回复人 7 萌萌点'
      : '您确定设置这个回复为最佳答案吗， 该操作将会为该回复人增加 7 萌萌点'
  )
  if (!res) {
    return
  }

  const result = await kunFetch<string>(
    `/topic/${Number(props.reply.topic_id)}/best-answer`,
    {
      method: 'PUT',
      body: {
        topic_id: Number(props.reply.topic_id),
        reply_id: Number(props.reply.id)
      }
    }
  )

  if (result) {
    useMessage(
      props.reply.is_best_answer ? '取消设置最佳答案成功' : '设置最佳答案成功',
      'success'
    )
    await refreshTopic()
  }
}
</script>

<template>
  <KunButton
    variant="light"
    :color="reply.is_best_answer ? 'warning' : 'default'"
    size="sm"
    :disabled="isDisabled"
    @click="handleUpdateTopicBestAnswer"
    class-name="whitespace-nowrap gap-2 justify-start"
  >
    <KunIcon class-name="text-lg" name="lucide:bookmark-check" />
    {{ reply.is_best_answer ? '取消设置最佳答案' : '将该回复设为最佳答案' }}
  </KunButton>
</template>
