<script setup lang="ts">
import type { Topic } from '#shared/utils/api/schemas'

const props = defineProps<{
  topic: Topic
  menu?: boolean
}>()

const { id, moemoepoint } = usePersistUserStore()
const isUpvoted = ref(props.topic.viewer?.has_upvoted ?? false)
const upvoteCount = ref(props.topic.upvote_count)

watch(
  () => props.topic.viewer?.has_upvoted,
  (value) => {
    isUpvoted.value = value ?? false
  }
)
watch(
  () => props.topic.upvote_count,
  (value) => {
    upvoteCount.value = value
  }
)

const { open } = useUpvoteModal()

const handleClickUpvote = async () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  if (id === Number(props.topic.author.id)) {
    useMessage(10241, 'warn')
    return
  }
  if (moemoepoint < 10) {
    useMessage(10242, 'warn')
    return
  }
  const pushed = await open({
    topicId: Number(props.topic.id),
    targetUserId: Number(props.topic.author.id)
  })
  if (pushed) {
    upvoteCount.value++
    isUpvoted.value = true
  }
}
</script>

<template>
  <KunButton
    v-if="menu"
    :variant="isUpvoted ? 'flat' : 'light'"
    :color="isUpvoted ? 'secondary' : 'default'"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    @click="handleClickUpvote"
  >
    <KunIcon class-name="text-lg" name="lucide:sparkles" />
    推话题
    <span v-if="upvoteCount" class="text-default-500 ml-auto">
      {{ upvoteCount }}
    </span>
  </KunButton>

  <KunTooltip v-else text="推话题">
    <KunReaction
      :toggle="false"
      :count="upvoteCount"
      label="推话题"
      @click="handleClickUpvote"
    >
      <template #icon>
        <KunIcon name="lucide:sparkles" class="text-warning" />
      </template>
    </KunReaction>
  </KunTooltip>
</template>
