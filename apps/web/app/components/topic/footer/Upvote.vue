<script setup lang="ts">
import type { Topic } from '#shared/utils/api/schemas'
import { KUN_MOEMOEPOINT } from '~/constants/moemoepoint'

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

const replaceTopic = inject<(topic: Topic) => void>('replaceTopic', () => {})
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
  if (moemoepoint < KUN_MOEMOEPOINT.upvoteSender) {
    useMessage(
      `您的萌萌点不足 ${KUN_MOEMOEPOINT.upvoteSender}, 无法使用推功能`,
      'warn'
    )
    return
  }
  const pushed = await open({
    topicId: props.topic.id,
    targetUserId: Number(props.topic.author.id)
  })
  if (pushed) {
    upvoteCount.value++
    isUpvoted.value = true
    replaceTopic({
      ...props.topic,
      upvote_count: upvoteCount.value,
      viewer: props.topic.viewer
        ? { ...props.topic.viewer, has_upvoted: true }
        : props.topic.viewer
    })
  }
}
</script>

<template>
  <template v-if="topic.viewer?.can_upvote">
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
</template>
