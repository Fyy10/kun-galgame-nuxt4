import catalogue from '../../../i18n/locales/zh-CN/problem.json'
import type { ClientProblem, FieldError } from './problem'

export const formatMessage = (
  template: string,
  params: Record<string, unknown> | undefined
): string =>
  template.replace(/\{([a-z][a-z0-9_]*)\}/g, (_match, name: string) => {
    const value = params?.[name]
    if (typeof value === 'number') {
      return value.toLocaleString('zh-CN')
    }
    if (Array.isArray(value)) {
      return value.map(String).join('、')
    }
    if (value === null || value === undefined) {
      return `{${name}}`
    }
    return String(value)
  })

export const problemMessage = (problem: ClientProblem): string => {
  if (problem.code && Object.hasOwn(catalogue.code, problem.code)) {
    return catalogue.code[problem.code as keyof typeof catalogue.code]
  }
  if (problem.kind === 'network') {
    return catalogue.client.network
  }
  if (problem.kind === 'timeout') {
    return catalogue.client.timeout
  }
  const statusKey = String(problem.status)
  if (Object.hasOwn(catalogue.status, statusKey)) {
    return catalogue.status[statusKey as keyof typeof catalogue.status]
  }
  return catalogue.client.default
}

export const fieldMessage = (error: FieldError): string => {
  if (!Object.hasOwn(catalogue.reason, error.reason)) {
    return catalogue.client.field
  }
  const variants =
    catalogue.reason[error.reason as keyof typeof catalogue.reason]
  const params = (error.params ?? {}) as Record<string, unknown>
  const present = (name: string) => {
    const value = params[name]
    return value !== null && value !== undefined
  }

  let bestKey: string | undefined
  let bestCount = -1
  for (const key of Object.keys(variants)) {
    if (key === 'default') {
      continue
    }
    const names = key.split('__')
    if (!names.every(present)) {
      continue
    }
    if (names.length > bestCount) {
      bestKey = key
      bestCount = names.length
    }
  }
  if (bestKey) {
    const picked: Record<string, unknown> = {}
    for (const name of bestKey.split('__')) {
      picked[name] = params[name]
    }
    const template = variants[bestKey as keyof typeof variants]
    return formatMessage(template, picked)
  }
  return variants.default
}
