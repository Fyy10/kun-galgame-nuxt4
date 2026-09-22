<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { Topic } from '#shared/utils/api/schemas'

const props = defineProps<{
  topic: Topic
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const replaceTopic = inject<(topic: Topic) => void>('replaceTopic', () => {})

const isAuthor = computed(() => !!id && String(id) === props.topic.author.id)
const isHidden = computed(() => props.topic.state === 'hidden')

type HideMode = 'hide' | 'unhide' | 'blocked' | 'none'

const mode = computed<HideMode>(() => {
  if (props.topic.viewer?.can_hide) {
    return 'hide'
  }
  if (props.topic.viewer?.can_unhide) {
    return 'unhide'
  }
  if (isHidden.value && isAuthor.value) {
    return 'blocked'
  }
  return 'none'
})

const confirmCopy = computed(() => {
  if (mode.value === 'unhide') {
    return {
      title: '确认取消隐藏这个话题吗',
      message: '取消隐藏后, 这个话题会重新回到列表与搜索中, 对所有人可见。'
    }
  }
  if (isAuthor.value) {
    return {
      title: '八嘎杂鱼笨蛋萝莉, 你要隐藏这个话题吗',
      message:
        '隐藏后话题会从列表与搜索中消失, 但您自己以及持有「查看隐藏话题」权限的管理人员仍然看得到它。您可以随时在个人主页的「已隐藏」中取消隐藏。'
    }
  }
  return {
    title: '确认隐藏这个话题吗',
    message:
      '隐藏后话题会从列表与搜索中消失, 仅作者本人与持有「查看隐藏话题」权限的管理人员仍可访问, 并且作者无法自行取消隐藏。'
  }
})

const isPending = ref(false)

const handleUpdateTopicHideStatus = async () => {
  const copy = confirmCopy.value
  const confirmed = await useComponentMessageStore().alert(
    copy.title,
    copy.message
  )
  if (!confirmed) {
    return
  }

  const wasHidden = isHidden.value
  isPending.value = true
  const result = await settle(
    api.PATCH('/topics/{topic_id}', {
      params: { path: { topic_id: props.topic.id } },
      body: { state: wasHidden ? 'published' : 'hidden' }
    })
  )
  isPending.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(wasHidden ? '取消隐藏话题成功' : '隐藏话题成功', 'success')
  replaceTopic(result.data)
}
</script>

<template>
  <KunButton
    v-if="mode === 'hide' || mode === 'unhide'"
    variant="light"
    :color="mode === 'unhide' ? 'primary' : 'danger'"
    size="sm"
    :loading="isPending"
    :disabled="isPending"
    @click="handleUpdateTopicHideStatus"
    class-name="whitespace-nowrap gap-2 justify-start"
  >
    <KunIcon
      class-name="text-lg"
      :name="mode === 'unhide' ? 'lucide:eye' : 'lucide:eye-off'"
    />
    {{ mode === 'unhide' ? '取消隐藏该话题' : '隐藏该话题' }}
  </KunButton>

  <div
    v-else-if="mode === 'blocked'"
    class="text-default-500 flex items-start gap-2 px-2 py-1 text-xs"
  >
    <KunIcon class-name="text-base shrink-0" name="lucide:shield-alert" />
    <span>该话题已被管理员隐藏, 无法自行取消</span>
  </div>
</template>
