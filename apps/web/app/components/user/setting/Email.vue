<script setup lang="ts">
const oauthProfileURL = computed(
  () => `${useRuntimeConfig().public.oauthFrontendUrl}/profile`
)
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div class="space-y-2">
      <span class="text-xl">更改邮箱</span>
      <p class="text-default-500 text-sm">
        邮箱由 {{ nextmoe.account }} 统一管理。修改邮箱需要二次验证
        (邮箱验证码), 请前往账号中心进行修改。修改后下次刷新页面即会同步至
        {{ kungal.titleShort }}。
      </p>
    </div>

    <div class="flex justify-end">
      <KunButton :href="oauthProfileURL" target="_blank">
        前往账号中心
      </KunButton>
    </div>
  </KunCard>
</template>
