<script setup lang="ts">
import { KunTooltip } from '#components'
import { applyTopicSource } from '~/composables/topic/applyTopicSource'
import { settle } from '#shared/utils/api/problem'
import type { Topic } from '#shared/utils/api/schemas'

const props = defineProps<{
  topic: Topic
  menu?: boolean
}>()

const api = useApiClient()
const isShowRewrite = computed(() => props.topic.viewer?.can_edit === true)

const rewriteTopic = async () => {
  const result = await settle(
    api.GET('/topics/{topic_id}/source', {
      params: { path: { topic_id: props.topic.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  applyTopicSource(result.data)
  await navigateTo('/edit/topic')
}
</script>

<template>
  <template v-if="isShowRewrite">
    <KunButton
      v-if="menu"
      variant="light"
      color="default"
      size="sm"
      class-name="w-full justify-start gap-2 whitespace-nowrap"
      @click="rewriteTopic"
    >
      <KunIcon class-name="text-lg" name="lucide:pencil" />
      重新编辑
    </KunButton>

    <KunTooltip v-else text="重新编辑">
      <KunReaction
        :toggle="false"
        icon="lucide:pencil"
        label="重新编辑"
        @click="rewriteTopic"
      />
    </KunTooltip>
  </template>
</template>
