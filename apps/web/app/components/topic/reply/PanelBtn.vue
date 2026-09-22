<script setup lang="ts">
import { createReplySchema, updateReplySchema } from '~/validations/topic'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const route = useRoute()
const topicId = computed(() => (route.params as { id: string }).id)

const tempReplyStore = useTempReplyStore()
const { isEdit, isReplyRewriting, replyRewrite } = storeToRefs(tempReplyStore)

const persistReplyStore = usePersistKUNGalgameReplyStore()
const { replyDraft } = storeToRefs(persistReplyStore)
const api = useApiClient()
const createKey = useIdempotencyKey()

const isPublishing = ref(false)

const handlePublish = async () => {
  if (isPublishing.value) {
    return
  }

  const body = {
    content_markdown: replyDraft.value.mainContent || ''
  }
  const parsed = createReplySchema.safeParse(body)
  if (!parsed.success) {
    const message = JSON.parse(parsed.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }

  const res = await useComponentMessageStore().alert('确认发布吗？')
  if (!res) {
    return
  }

  isPublishing.value = true
  const result = await settle(
    api.POST('/topics/{topic_id}/replies', {
      params: {
        path: { topic_id: topicId.value },
        header: {
          'Idempotency-Key': createKey.take(
            `/topics/${topicId.value}/replies`,
            body
          )
        }
      },
      body
    })
  )
  isPublishing.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  createKey.clear()
  isEdit.value = false
  tempReplyStore.setSuccessfulReply({ data: result.data, type: 'created' })
  persistReplyStore.resetReplyDraft()
  useMessage(10243, 'success')
}

const handleRewrite = async () => {
  if (isPublishing.value) {
    return
  }

  const replyId = replyRewrite.value!.id
  const body = {
    content_markdown: replyRewrite.value!.mainContent || ''
  }
  const parsed = updateReplySchema.safeParse(body)
  if (!parsed.success) {
    const message = JSON.parse(parsed.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }
  const res = await useComponentMessageStore().alert('确定提交编辑吗?')
  if (!res) {
    return
  }

  isPublishing.value = true
  const result = await settle(
    api.PATCH('/replies/{reply_id}', {
      params: { path: { reply_id: replyId } },
      body
    })
  )
  isPublishing.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(10244, 'success')
  tempReplyStore.setSuccessfulReply({ data: result.data, type: 'updated' })
  tempReplyStore.resetRewriteReplyData()
  isEdit.value = false
}

const handleCancel = () => {
  isEdit.value = false
  tempReplyStore.resetRewriteReplyData()
}
</script>

<template>
  <div class="flex justify-between gap-1">
    <KunTooltip class-name="flex" text="设置面板帮助" position="bottom">
      <KunLink to="/doc/reply-panel-help" size="sm" underline="hover">
        回复面板使用帮助
        <KunIcon name="lucide:circle-help" />
      </KunLink>
    </KunTooltip>

    <div class="space-x-1">
      <KunButton color="danger" variant="light" @click="handleCancel">
        取消
      </KunButton>

      <KunButton
        :loading="isPublishing"
        v-if="!isReplyRewriting"
        @click="handlePublish"
      >
        确认发布
      </KunButton>

      <KunButton
        :loading="isPublishing"
        v-if="isReplyRewriting"
        @click="handleRewrite"
      >
        确定编辑
      </KunButton>
    </div>
  </div>
</template>
