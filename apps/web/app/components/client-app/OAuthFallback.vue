<script setup lang="ts">
const route = useRoute()
const router = useRouter()

onMounted(() => {
  if (Object.keys(route.query).length) {
    router.replace({ path: route.path })
  }
})
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
        <h1 class="text-xl font-bold">请在 App 中完成登录</h1>
        <p class="text-default-500 text-sm">
          这是 {{ kungal.name }} App
          的登录回调页面。看到这里说明登录结果没能交回
          App：可能这台设备还没有安装 App，或者 App 版本过旧。请安装或更新 App
          后，在 App 内重新登录。
        </p>
      </div>

      <div class="flex flex-col gap-3">
        <KunButton color="primary" size="lg" full-width href="/app">
          下载 {{ kungal.name }} App
        </KunButton>
        <KunButton variant="flat" size="lg" full-width href="/">
          返回首页
        </KunButton>
      </div>
    </KunCard>
  </div>
</template>
