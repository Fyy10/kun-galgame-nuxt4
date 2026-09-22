<script setup lang="ts">
import type { Reply, Topic } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import type { ComputedRef } from 'vue'

const props = defineProps<{
  reply: Reply
}>()

const api = useApiClient()
const pageTopic = inject<ComputedRef<Topic>>('pageTopic')
const replaceTopic = inject<(topic: Topic) => void>('replaceTopic', () => {})
const canPin = computed(() => pageTopic?.value.viewer?.can_pin_reply === true)

const handleUpdateReplyPin = async () => {
  const res = await useComponentMessageStore().alert(
    props.reply.is_pinned
      ? '您确定取消置顶该回复吗'
      : '您确定将该回复置顶吗? 置顶可以随时设置和取消'
  )
  if (!res) {
    return
  }

  const topicId = props.reply.topic_id
  const result = await settle(
    props.reply.is_pinned
      ? api.DELETE('/topics/{topic_id}/pinned-reply', {
          params: { path: { topic_id: topicId } }
        })
      : api.PUT('/topics/{topic_id}/pinned-reply', {
          params: { path: { topic_id: topicId } },
          body: { reply_id: props.reply.id }
        })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(
    props.reply.is_pinned ? '取消置顶回复成功' : '置顶回复成功',
    'success'
  )
  replaceTopic(result.data)
}
</script>

<template>
  <KunButton
    v-if="canPin"
    variant="light"
    :color="reply.is_pinned ? 'warning' : 'default'"
    size="sm"
    @click="handleUpdateReplyPin"
    class-name="whitespace-nowrap gap-2 justify-start"
  >
    <KunIcon class-name="text-lg" name="lucide:pin" />
    {{ reply.is_pinned ? '取消置顶回复' : '置顶回复' }}
  </KunButton>
</template>
