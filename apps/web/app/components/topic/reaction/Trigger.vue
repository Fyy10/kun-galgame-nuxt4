<script setup lang="ts">
const { toggle, mineKeys, list, topicId, replyId } = inject(reactionsKey)!

const picker = ref<{ close: () => void } | null>(null)
const historyOpen = ref(false)

const subject = topicId ? 'topic' : replyId ? 'reply' : undefined

const total = computed(() => list.value.reduce((sum, r) => sum + r.count, 0))

const onSelect = (key: string) => {
  picker.value?.close()
  toggle(key)
}

const onViewHistory = () => {
  picker.value?.close()
  historyOpen.value = true
}
</script>

<template>
  <KunPopover ref="picker" position="top-start" opaque>
    <template #trigger>
      <KunReaction :toggle="false" icon="lucide:smile-plus" label="表态" />
    </template>

    <TopicReactionPicker
      :mine-keys="mineKeys"
      :subject="subject"
      :total="total"
      @select="onSelect"
      @view-history="onViewHistory"
    />
  </KunPopover>

  <TopicReactionHistoryModal
    v-if="historyOpen"
    v-model="historyOpen"
    :topic-id="topicId"
    :reply-id="replyId"
  />
</template>
