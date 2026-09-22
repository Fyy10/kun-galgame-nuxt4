<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    favorited: boolean
    count: number
    endpoint?: string
    body?: Record<string, unknown>
    action?: (next: boolean) => Promise<boolean>
    messages?: [string | number, string | number]
    label?: string
    tooltip?: string
    size?: 'sm' | 'md' | 'lg'
  }>(),
  {
    label: '收藏',
    tooltip: '收藏',
    size: 'md',
    body: undefined,
    messages: undefined,
    endpoint: undefined,
    action: undefined
  }
)

const emits = defineEmits<{ changed: [favorited: boolean] }>()

const { id } = usePersistUserStore()
const isFavorited = ref(props.favorited)
const favoriteCount = ref(props.count)
const pending = ref(false)

watch(
  () => props.favorited,
  (v) => (isFavorited.value = v)
)
watch(
  () => props.count,
  (v) => (favoriteCount.value = v)
)

const revert = (next: boolean) => {
  isFavorited.value = !next
  favoriteCount.value += next ? -1 : 1
}

const onChange = async (next: boolean) => {
  if (!id) {
    useAuthModal().open()
    revert(next)
    return
  }
  pending.value = true
  let ok = false
  if (props.action) {
    ok = await props.action(next)
  } else {
    const result = await kunFetch(props.endpoint!, {
      method: 'PUT',
      ...(props.body ? { body: props.body } : {})
    })
    ok = Boolean(result)
  }
  pending.value = false
  if (!ok) {
    revert(next)
    return
  }
  if (props.messages) {
    useMessage(next ? props.messages[0] : props.messages[1], 'success')
  }
  emits('changed', next)
}
</script>

<template>
  <KunTooltip :text="tooltip">
    <span class="flex">
      <KunReaction
        v-model="isFavorited"
        v-model:count="favoriteCount"
        :disabled="pending"
        :size="size"
        icon="lucide:heart"
        color="danger"
        :label="label"
        @change="onChange"
      />
    </span>
  </KunTooltip>
</template>
