<script setup lang="ts">
useKunDisableSeo('请在 App 中完成登录')
useHead({ meta: [{ name: 'referrer', content: 'no-referrer' }] })
</script>

<template>
  <ClientAppOAuthFallback />
</template>
