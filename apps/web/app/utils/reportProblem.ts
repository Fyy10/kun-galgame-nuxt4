import { fieldMessage, problemMessage } from '#shared/utils/api/message'
import type { ClientProblem } from '#shared/utils/api/problem'

export const reportProblem = (problem: ClientProblem) => {
  if (import.meta.server) {
    return
  }
  useMessage(problemMessage(problem), 'error')
  for (const error of problem.errors) {
    useMessage(fieldMessage(error), 'error')
  }
}
