<script setup lang="ts">
const open = defineModel<boolean>({ required: true })

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
  <KunModal
    v-model="open"
    inner-class-name="max-w-md overflow-visible"
    :is-dismissable="false"
  >
    <div class="space-y-6 p-2 select-none">
      <div class="flex flex-col items-center gap-3">
        <KunImage src="/favicon.webp" class-name="h-12 w-12 rounded-2xl" />
        <h2 class="text-xl font-bold">欢迎来到 {{ kungal.titleShort }}</h2>
        <p class="text-default-500 text-center text-sm">
          登录或注册以解锁完整功能。账号统一由
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
    </div>
  </KunModal>
</template>
