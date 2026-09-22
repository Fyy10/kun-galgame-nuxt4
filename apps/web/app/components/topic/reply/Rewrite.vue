<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  reply: Reply
}>()

const { setRewriteData } = useTempReplyStore()
const { isEdit } = storeToRefs(useTempReplyStore())
const api = useApiClient()
const isShowRewrite = computed(() => props.reply.viewer?.can_edit === true)

const handleClickRewrite = async () => {
  const result = await settle(
    api.GET('/replies/{reply_id}/source', {
      params: { path: { reply_id: props.reply.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  setRewriteData({
    id: result.data.reply_id,
    mainContent: result.data.content_markdown
  })
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
