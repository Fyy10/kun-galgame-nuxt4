<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'

const props = defineProps<{
  reply: Reply
}>()

const { id } = usePersistUserStore()
const canPinReply = useCan('reply.pin')
const topicUserId = inject<number>('topicUserId')
const refreshTopic = inject<() => Promise<unknown>>('refreshTopic', () =>
  Promise.resolve()
)

const isDisabled = !canPinReply.value && topicUserId !== id

const handleUpdateReplyPin = async () => {
  const res = await useComponentMessageStore().alert(
    props.reply.is_pinned
      ? '您确定取消置顶该回复吗'
      : '您确定将该回复置顶吗? 置顶可以随时设置和取消'
  )
  if (!res) {
    return
  }

  const result = await kunFetch<string>(
    `/topic/${Number(props.reply.topic_id)}/reply/pin`,
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
      props.reply.is_pinned ? '取消置顶回复成功' : '置顶回复成功',
      'success'
    )
    await refreshTopic()
  }
}
</script>

<template>
  <KunButton
    variant="light"
    :color="reply.is_pinned ? 'warning' : 'default'"
    size="sm"
    :disabled="isDisabled"
    @click="handleUpdateReplyPin"
    class-name="whitespace-nowrap gap-2 justify-start"
  >
    <KunIcon class-name="text-lg" name="lucide:pin" />
    {{ reply.is_pinned ? '取消置顶回复' : '置顶回复' }}
  </KunButton>
</template>
