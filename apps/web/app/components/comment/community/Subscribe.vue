<script setup lang="ts">
const props = defineProps<{
  following: boolean
  submit: (next: boolean) => Promise<void>
}>()

const pending = ref(false)

const toggle = async () => {
  pending.value = true
  await props.submit(!props.following)
  pending.value = false
}
</script>

<template>
  <KunButton
    variant="light"
    size="sm"
    :color="following ? 'primary' : 'default'"
    :loading="pending"
    @click="toggle"
  >
    <KunIcon :name="following ? 'lucide:bell' : 'lucide:bell-plus'" />
    {{ following ? '已关注' : '关注' }}
  </KunButton>
</template>
