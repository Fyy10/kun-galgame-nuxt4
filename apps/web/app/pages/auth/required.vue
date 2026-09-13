<script setup lang="ts">
useKunDisableSeo('需要登录')

const route = useRoute()
const redirectTo = computed(() => {
  const r = (route.query.redirect as string) || ''
  return r.startsWith('/') && !r.startsWith('//') ? r : ''
})

const isJumping = ref(false)

const handleLogin = async () => {
  isJumping.value = true
  await startOAuthLogin()
}

const handleRegister = async () => {
  isJumping.value = true
  await startOAuthRegister()
}
</script>

<template>
  <div class="flex min-h-[calc(100dvh-12rem)] items-center justify-center p-4">
    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      class-name="w-full max-w-md"
      content-class="space-y-6"
    >
      <div class="flex flex-col items-center gap-3 text-center">
        <KunImage src="/favicon.webp" class-name="h-14 w-14 rounded-2xl" />
        <h1 class="text-xl font-bold">需要登录</h1>
        <p class="text-default-500 text-sm">
          该页面需要登录后才能访问。登录或注册以解锁完整功能，账号统一由
          <span class="text-default-700 font-medium">{{
            nextmoe.account
          }}</span>
          管理。
        </p>
      </div>

      <div class="flex flex-col gap-3">
        <KunButton
          color="primary"
          size="lg"
          full-width
          :disabled="isJumping"
          @click="handleLogin"
        >
          {{ isJumping ? '跳转中...' : '登录' }}
        </KunButton>
        <KunButton
          variant="flat"
          color="primary"
          size="lg"
          full-width
          :disabled="isJumping"
          @click="handleRegister"
        >
          {{ isJumping ? '跳转中...' : '注册新账号' }}
        </KunButton>
      </div>

      <div class="flex items-center justify-center gap-4 text-sm">
        <KunLink :to="redirectTo || '/'" underline="hover" color="default">
          {{ redirectTo ? '我已登录，返回上一页' : '返回首页' }}
        </KunLink>
      </div>

      <div
        class="text-default-400 flex items-center justify-center gap-1.5 text-xs"
      >
        <KunImage
          :src="nextmoe.logo"
          :alt="nextmoe.account"
          class-name="ring-default-200 size-5 shrink-0 rounded-full ring-1"
        />
        点击按钮将跳转至 {{ nextmoe.account }}
      </div>
    </KunCard>
  </div>
</template>
