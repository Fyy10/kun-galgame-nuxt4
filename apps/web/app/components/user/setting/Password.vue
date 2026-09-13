<script setup lang="ts">
const oauthProfileURL = computed(
  () => `${useRuntimeConfig().public.oauthFrontendUrl}/profile`
)
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div class="space-y-2">
      <span class="text-xl">更改密码</span>
      <p class="text-default-500 text-sm">
        密码由 {{ nextmoe.account }} 统一管理。修改密码需要验证旧密码,
        请前往账号中心进行修改。忘记密码同样在那里处理。
      </p>
    </div>

    <div class="flex justify-end">
      <KunButton :href="oauthProfileURL" target="_blank">
        前往账号中心
      </KunButton>
    </div>
  </KunCard>
</template>
