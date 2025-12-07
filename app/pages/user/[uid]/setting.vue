<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

defineProps<{
  user: UserInfo
}>()

useKunDisableSeo('信息设置')
</script>

<template>
  <UserSetting :user="user" />
</template>
