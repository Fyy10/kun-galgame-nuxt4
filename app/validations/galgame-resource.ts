import { z } from 'zod'

export const getGalgameResourceSchema = z.object({
  page: z.coerce.number<number>().min(1).max(9999999),
  limit: z.coerce.number<number>().min(1).max(50)
})

export const getGalgameResourceDetailSchema = z.object({
  resourceId: z.coerce.number<number>().min(1).max(9999999)
})
