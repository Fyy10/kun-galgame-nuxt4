import type { GalgameResourceStoreTemp } from '~/store/types/galgame/resource'
import { parseResourceSize } from '~~/shared/utils/resourceSize'
import {
  RESOURCE_TYPE_LABELS,
  LANGUAGE_LABELS,
  PLATFORM_LABELS,
  RUNTIME_LABELS,
  hasRuntimeAxis
} from '~~/shared/utils/galgameResourceVocab'

export const checkGalgameResourcePublish = (link: GalgameResourceStoreTemp) => {
  if (!RESOURCE_TYPE_LABELS[link.type]) {
    useMessage(10556, 'warn')
    return false
  }

  if (!link.link.length || link.link.length > 20) {
    useMessage(10557, 'warn')
    return false
  }

  for (const l of link.link) {
    if (l.trim().length > 1007) {
      useMessage(10558, 'warn')
      return false
    }

    if (!isValidURL(l.trim())) {
      useMessage(10559, 'warn')
      return false
    }
  }

  if (
    !link.languages.length ||
    link.languages.some((k) => !LANGUAGE_LABELS[k])
  ) {
    useMessage(10560, 'warn')
    return false
  }

  if (
    !link.platforms.length ||
    link.platforms.some((k) => !PLATFORM_LABELS[k])
  ) {
    useMessage(10561, 'warn')
    return false
  }

  if (hasRuntimeAxis(link.type)) {
    if (
      !link.runtimes.length ||
      link.runtimes.some((k) => !RUNTIME_LABELS[k])
    ) {
      useMessage(10570, 'warn')
      return false
    }
  }

  if (!parseResourceSize(link.size)) {
    useMessage(10562, 'warn')
    return false
  }

  if (link.code.length > 1007) {
    useMessage(10563, 'warn')
    return false
  }

  if (link.password.length > 1007) {
    useMessage(10564, 'warn')
    return false
  }

  if (link.note.length > 10000) {
    useMessage(10565, 'warn')
    return false
  }

  return true
}
