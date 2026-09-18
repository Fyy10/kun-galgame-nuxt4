// @vitest-environment node
import { fileURLToPath } from 'node:url'
import { ESLint } from 'eslint'
import { describe, expect, it } from 'vitest'

const webRoot = fileURLToPath(new URL('../..', import.meta.url))
const eslint = new ESLint({ cwd: webRoot })

const hits = async (code: string, filePath: string) => {
  const [result] = await eslint.lintText(code, { filePath })
  return (result?.messages ?? []).filter(
    (message) => message.ruleId === 'no-restricted-syntax'
  )
}

describe('F6 v1 literals', () => {
  it('reports /api/v1 and /v1 path literals under app/', async () => {
    const appFile = `${webRoot}/app/example.ts`
    expect(
      (await hits("const path = '/api/v1/topics'\n", appFile)).length
    ).toBeGreaterThan(0)
    expect(
      (await hits('const path = `/v1/topics`\n', appFile)).length
    ).toBeGreaterThan(0)
    expect(
      (await hits('const path = `${base}/api/v1/topics`\n', appFile)).length
    ).toBeGreaterThan(0)
  })

  it('does not report the same code in the typed client', async () => {
    const clientFile = `${webRoot}/shared/utils/api/client.ts`
    expect(
      (await hits("const path = '/api/v1/topics'\n", clientFile)).length
    ).toBe(0)
    expect((await hits('const path = `/v1/topics`\n', clientFile)).length).toBe(
      0
    )
    expect(
      (await hits('const path = `${base}/api/v1/topics`\n', clientFile)).length
    ).toBe(0)
  })

  it('does not report a sticker-site absolute URL under app/', async () => {
    const appFile = `${webRoot}/app/example.ts`
    expect(
      (
        await hits(
          "const path = 'https://sticker.kungal.com/api/v1/avatar-pool'\n",
          appFile
        )
      ).length
    ).toBe(0)
  })
})
