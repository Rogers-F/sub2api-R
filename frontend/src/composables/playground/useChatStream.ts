import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { chatCompletion, chatCompletionStream } from '@/api/playground'
import { MESSAGE_ROLES, MESSAGE_STATUS, STREAM_STATUS } from '@/constants/playground'
import { usePlaygroundStore } from '@/stores/playground'
import { useAppStore } from '@/stores/app'
import {
  buildMessageContent,
  createLoadingAssistantMessage,
  createMessage
} from '@/utils/playground/messageHelpers'
import { buildApiPayload } from '@/utils/playground/payloadBuilder'

export function useChatStream() {
  const store = usePlaygroundStore()
  const appStore = useAppStore()
  const { t } = useI18n()

  const abortController = ref<AbortController | null>(null)

  async function sendMessage(userText: string) {
    if (!store.apiKey) {
      appStore.showError(t('playground.errors.noKeySelected'))
      return
    }
    if (!store.inputs.model) {
      appStore.showError(t('playground.errors.noModelSelected'))
      return
    }

    const userContent = buildMessageContent(
      userText,
      store.inputs.imageUrls,
      store.inputs.imageEnabled
    )
    store.addMessage(createMessage(MESSAGE_ROLES.USER, userContent))
    store.addMessage(createLoadingAssistantMessage())
    store.setStatus(STREAM_STATUS.STREAMING)

    const payload = buildApiPayload([...store.messages], store.inputs, store.parameterEnabled)
    abortController.value = new AbortController()

    if (store.inputs.stream) {
      await runStream(payload)
    } else {
      await runNonStream(payload)
    }
  }

  async function runNonStream(payload: ReturnType<typeof buildApiPayload>) {
    try {
      const result = (await chatCompletion(
        store.apiKey,
        payload,
        abortController.value?.signal
      )) as { choices?: Array<{ message?: { content?: string; reasoning_content?: string } }> }
      const choice = result.choices?.[0]
      store.patchLastMessage({
        content: choice?.message?.content ?? '',
        reasoningContent: choice?.message?.reasoning_content ?? '',
        status: MESSAGE_STATUS.COMPLETE
      })
      store.setStatus(STREAM_STATUS.COMPLETE)
    } catch (e: any) {
      handleError(e?.message || String(e))
    } finally {
      abortController.value = null
    }
  }

  async function runStream(payload: ReturnType<typeof buildApiPayload>) {
    let acc = ''
    let reasoningAcc = ''
    await chatCompletionStream(
      store.apiKey,
      payload,
      {
        onChunk: (_raw, parsed: any) => {
          const delta = parsed?.choices?.[0]?.delta
          if (!delta) return
          if (typeof delta.content === 'string' && delta.content) {
            acc += delta.content
          }
          // OpenAI Responses / DeepSeek 用 reasoning_content；旧字段 reasoning 也兼容
          const r = delta.reasoning_content ?? delta.reasoning
          if (typeof r === 'string' && r) {
            reasoningAcc += r
          }
          store.patchLastMessage({
            content: acc,
            reasoningContent: reasoningAcc,
            status: MESSAGE_STATUS.INCOMPLETE
          })
        },
        onDone: () => {
          store.patchLastMessage({ status: MESSAGE_STATUS.COMPLETE })
          store.setStatus(STREAM_STATUS.COMPLETE)
        },
        onError: (err) => {
          handleError(
            t('playground.errors.httpError', { status: err.status || 0, message: err.message })
          )
        }
      },
      abortController.value?.signal
    )
    abortController.value = null
  }

  function stopGeneration() {
    if (abortController.value) {
      abortController.value.abort()
      abortController.value = null
    }
    store.patchLastMessage({ status: MESSAGE_STATUS.COMPLETE })
    store.setStatus(STREAM_STATUS.IDLE)
    appStore.showInfo(t('playground.errors.requestAborted'))
  }

  function handleError(message: string) {
    store.patchLastMessage({
      status: MESSAGE_STATUS.ERROR,
      errorMessage: message
    })
    store.setStatus(STREAM_STATUS.ERROR, message)
    appStore.showError(message)
  }

  return {
    sendMessage,
    stopGeneration,
    isStreaming: () => store.status === STREAM_STATUS.STREAMING
  }
}
