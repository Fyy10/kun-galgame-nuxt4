<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'

const props = defineProps<{
  message: ChatMessage
  isSent: boolean
}>()

const emit = defineEmits<{
  (event: 'context-menu', payload: { event: MouseEvent; message: ChatMessage }): void
}>()

const isMobile = useMediaQuery('(max-width: 640px)')
const canRecall = computed(() => props.isSent && !props.message.isRecall)
const recallText = computed(
  () => `${props.message.sender.name}撤回了一条消息`
)
const recallCursorClass = computed(() => {
  if (!canRecall.value) {
    return ''
  }

  return isMobile.value ? 'cursor-pointer' : 'cursor-context-menu'
})

const handleContextMenu = (event: MouseEvent) => {
  if (!canRecall.value || isMobile.value) {
    return
  }
  event.stopPropagation()
  emit('context-menu', { event, message: props.message })
}

const handleClick = (event: MouseEvent) => {
  if (!canRecall.value || !isMobile.value) {
    return
  }
  event.stopPropagation()
  emit('context-menu', { event, message: props.message })
}
</script>

<template>
  <div
    class="flex w-full"
    :class="[
      message.isRecall ? 'items-center justify-center py-2' : 'items-end gap-2',
      message.isRecall ? '' : isSent ? 'flex-row-reverse' : 'flex-row'
    ]"
  >
    <template v-if="message.isRecall">
      <span
        class="bg-default-100 text-default-500 text-xs sm:text-sm rounded-full px-3 py-1"
      >
        {{ recallText }}
      </span>
    </template>

    <template v-else>
      <KunAvatar
        :disable-floating="true"
        :user="message.sender"
        class="mb-auto"
      />

      <div
        class="relative max-w-[75%] rounded-lg border p-3 transition-colors"
        :class="[
          isSent
            ? 'bg-primary/20 border-primary/20'
            : 'bg-background border-default-300',
          recallCursorClass
        ]"
        @contextmenu.prevent="handleContextMenu"
        @click="handleClick"
      >
        <div class="flex items-end">
          <span
            class="text-sm font-medium"
            :class="isSent ? 'text-primary' : 'text-secondary'"
          >
            {{ message.sender.name }}
          </span>
        </div>

        <div class="mt-1 text-sm leading-relaxed">
          <span>{{ message.content }}</span>
          <span class="text-default-500 ml-2 text-xs">
            {{ formatTimeDifference(message.created) }}
          </span>
        </div>
      </div>
    </template>
  </div>
</template>
