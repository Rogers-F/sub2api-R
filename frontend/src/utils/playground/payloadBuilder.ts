import { MESSAGE_STATUS } from '@/constants/playground'
import type {
  ChatCompletionPayload,
  ParameterEnabled,
  PlaygroundInputs,
  PlaygroundMessage
} from '@/types/playground'

export function buildApiPayload(
  messages: PlaygroundMessage[],
  inputs: PlaygroundInputs,
  parameterEnabled: ParameterEnabled
): ChatCompletionPayload {
  const ready = messages.filter(
    (m) => m.status !== MESSAGE_STATUS.LOADING && m.status !== MESSAGE_STATUS.ERROR
  )

  const payload: ChatCompletionPayload = {
    model: inputs.model,
    messages: ready.map((m) => ({ role: m.role, content: m.content })),
    stream: inputs.stream
  }

  if (parameterEnabled.temperature) payload.temperature = inputs.temperature
  if (parameterEnabled.top_p) payload.top_p = inputs.top_p
  if (parameterEnabled.max_tokens) payload.max_tokens = inputs.max_tokens
  if (parameterEnabled.frequency_penalty)
    payload.frequency_penalty = inputs.frequency_penalty
  if (parameterEnabled.presence_penalty)
    payload.presence_penalty = inputs.presence_penalty
  if (parameterEnabled.seed && inputs.seed !== null) payload.seed = inputs.seed

  return payload
}
