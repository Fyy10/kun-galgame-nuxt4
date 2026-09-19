<script setup lang="ts">
import { KunTooltip } from '#components'
import { toTopicAccessRoles, toTopicAccessScope } from '~/constants/topic'
import type { Topic } from '#shared/utils/api/schemas'

const props = defineProps<{
  topic: Topic
  menu?: boolean
}>()

const {
  id,
  title,
  content,
  category,
  section,
  isNSFW,
  coverImages,
  accessScope,
  accessRoles,
  accessUserIds,
  isTopicRewriting
} = storeToRefs(useTempEditStore())
const { id: userId } = usePersistUserStore()
const canEditAnyTopic = useCan('topic.edit_any')
const isShowRewrite = computed(
  () => userId === Number(props.topic.author.id) || canEditAnyTopic.value
)

const rewriteTopic = async () => {
  const detail = await kunFetch<TopicDetail>(
    `/topic/${Number(props.topic.id)}`,
    {
      method: 'GET',
      query: { topic_id: Number(props.topic.id) }
    }
  )
  if (!detail) {
    return
  }
  id.value = detail.id
  title.value = detail.title
  content.value = detail.content_markdown
  category.value = detail.category
  section.value = detail.section ?? []
  isNSFW.value = !!detail.is_nsfw
  coverImages.value = detail.cover_images ?? []
  accessScope.value = toTopicAccessScope(detail.access_scope)
  accessRoles.value = toTopicAccessRoles(detail.access_grants?.roles)
  accessUserIds.value = detail.access_grants?.user_ids ?? []
  isTopicRewriting.value = true

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
