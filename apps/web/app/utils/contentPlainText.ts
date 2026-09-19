import {
  documentHeadings as headingsOf,
  documentPlainText as toPlain
} from '#shared/utils/content/plainText'
import type { ContentDocument } from '#shared/utils/api/schemas'
import { deletedUserName } from './userRef'

export const contentPlainText = (
  input: ContentDocument | readonly unknown[] | null | undefined
) => toPlain(input, deletedUserName)

export const contentHeadings = (
  input: ContentDocument | readonly unknown[] | null | undefined
) => headingsOf(input, deletedUserName)
