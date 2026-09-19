<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'

const props = defineProps<{
  reply: Reply
}>()

const { setRewriteData } = useTempReplyStore()
const { isEdit } = storeToRefs(useTempReplyStore())
const { id } = usePersistUserStore()

const canEditAnyReply = useCan('reply.edit_any')
const isShowRewrite = computed(
  () => id === Number(props.reply.author.id) || canEditAnyReply.value
)

const handleClickRewrite = async () => {
  const detail = await kunFetch<TopicReply>(
    `/topic/${Number(props.reply.topic_id)}/reply/detail`,
    {
      method: 'GET',
      query: { replyId: Number(props.reply.id) }
    }
  )
  if (!detail) {
    return
  }
  setRewriteData(detail)
  isEdit.value = true
}
</script>

<template>
  <KunButton
    v-if="isShowRewrite"
    variant="light"
    color="default"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    @click="handleClickRewrite"
  >
    <KunIcon class-name="text-lg" name="lucide:pencil" />
    重新编辑
  </KunButton>
</template>
