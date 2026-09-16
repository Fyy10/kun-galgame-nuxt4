<script setup lang="ts">
const props = defineProps<{
  threadId: number
  subscription: CommunityThreadState | null
  submit: (level: CommunityNotificationLevel) => Promise<void>
}>()

const pending = ref(false)

const subscribed = computed(() => props.subscription?.subscribed ?? false)

// The community service has four levels, but its unread listing only ever asks
// "is this muted?" — normal, tracking and watching all count the same. Offering
// four buttons for one distinction would be three lies.
const toggle = async () => {
  pending.value = true
  await props.submit(subscribed.value ? 0 : 3)
  pending.value = false
}
</script>

<template>
  <!--
    A wall nobody has commented on has no thread yet, and following a thread
    that does not exist is not a thing the community service can do — the first
    comment is what creates it.
  -->
  <KunButton
    v-if="threadId"
    variant="light"
    size="sm"
    :color="subscribed ? 'primary' : 'default'"
    :loading="pending"
    @click="toggle"
  >
    <KunIcon :name="subscribed ? 'lucide:bell' : 'lucide:bell-plus'" />
    {{ subscribed ? '已关注' : '关注' }}
  </KunButton>
</template>
