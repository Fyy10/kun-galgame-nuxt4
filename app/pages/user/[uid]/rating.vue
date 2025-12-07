<script setup lang="ts">
const props = defineProps<{
  user: UserInfo
}>()

useKunDisableSeo(`${props.user.name} 的评分`)

useHead({
  link: [
    {
      rel: 'canonical',
      href: `${kungal.domain.main}/user/${props.user.id}/rating`
    }
  ]
})
</script>

<template>
  <UserRating :uid="user.id" />
</template>
