import { z } from 'zod'

const optionText = z
  .string()
  .min(1, '选项内容不能为空')
  .max(100, '选项内容最多100个字符')

const optionId = z.string().regex(/^[0-9]+$/, '选项编号不正确')

const pollSchema = z.object({
  title: z
    .string()
    .min(1, '投票标题不能为空')
    .max(100, '投票标题最多100个字符'),
  description: z.string().max(500, '投票描述最多500个字符').default(''),
  choice_type: z.enum(['single', 'multiple'], {
    message: '投票类型必须是单选或多选'
  }),
  min_choice: z.coerce.number<number>().int().min(1).max(20),
  max_choice: z.coerce.number<number>().int().min(1).max(20),
  closes_at: z.iso.datetime().optional(),
  result_visibility: z.enum(['always', 'after_vote', 'after_deadline'], {
    message: '结果可见性设置不正确'
  }),
  is_anonymous: z.boolean(),
  can_change_vote: z.boolean()
})

export const createPollSchema = pollSchema.extend({
  options: z
    .array(z.object({ text: optionText }))
    .min(2, '投票至少需要2个选项')
    .max(20, '投票最多只能有20个选项')
})

export const updatePollSchema = pollSchema.extend({
  option_changes: z.object({
    add: z.array(z.object({ text: optionText })).max(20),
    update: z
      .array(z.object({ option_id: optionId, text: optionText }))
      .max(20),
    remove: z.array(optionId).max(20)
  })
})
