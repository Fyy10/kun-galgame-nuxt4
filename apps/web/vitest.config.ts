import { defineVitestConfig } from '@nuxt/test-utils/config'

export default defineVitestConfig({
  test: {
    // Two environments coexist. This is the default, for pure ts/js utils;
    // component and composable specs opt in per file with the docblock
    // `// @vitest-environment nuxt`. That pragma is load-bearing — delete it
    // and the spec silently drops to happy-dom, where mountSuspended dies with
    // "Cannot read properties of undefined (reading 'vueApp')".
    environment: 'happy-dom',
    globals: true,
    include: [
      'app/**/*.spec.ts',
      'shared/**/*.spec.ts',
      'server/**/*.spec.ts',
      'tests/**/*.spec.ts'
    ],
    exclude: ['**/node_modules/**', '**/dist/**', '**/.nuxt/**']
  }
})
