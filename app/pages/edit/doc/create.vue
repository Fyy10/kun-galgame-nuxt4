<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

useKunSeoMeta({ title: '创建文档' })
</script>

<template>
  <EditDocLayout mode="create" />
</template>
