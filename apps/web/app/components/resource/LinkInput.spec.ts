// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import ResourceLinkInput from './LinkInput.vue'

describe('ResourceLinkInput', () => {
  it('labels each filled URL with the site netdisk kind', async () => {
    const wrapper = await mountSuspended(ResourceLinkInput, {
      props: {
        modelValue:
          'https://pan.baidu.com/s/aaa, https://pan.quark.cn/s/bbb, https://www.alipan.com/s/ccc'
      }
    })
    const text = wrapper.text()
    expect(text).toContain('百度网盘')
    expect(text).toContain('夸克网盘')
    expect(text).toContain('阿里云盘')
    expect(text).toContain('https://pan.baidu.com/s/aaa')
  })

  it('updates the kind when the link field changes', async () => {
    const wrapper = await mountSuspended(ResourceLinkInput, {
      props: { modelValue: 'https://www.123pan.com/s/xxx' }
    })
    expect(wrapper.text()).toContain('123盘')
    await wrapper.setProps({
      modelValue: 'https://cloud.189.cn/t/yyy'
    })
    expect(wrapper.text()).toContain('天翼云盘')
    expect(wrapper.text()).not.toContain('123盘')
  })
})
