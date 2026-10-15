import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { AppError } from '../types/domain'
import type { SmsCard } from '../types/sms'
import type { SMSMessageDTO, SmsThreadVM } from '../types/view-model'
import {
  smsService,
  type SmsDeleteThreadPayload,
  type SmsMarkThreadReadPayload,
  type SmsSendPayload,
  type SmsThreadQueryParams
} from '../services/sms'

export const useSMSStore = defineStore('sms', () => {
  const cards = ref<SmsCard[]>([])
  const threads = ref<SmsThreadVM[]>([])
  const threadMessages = ref<SMSMessageDTO[]>([])

  const loading = ref(false)
  const lastOkAt = ref<number | null>(null)
  const error = ref<AppError | null>(null)

  async function fetchCards() {
    const result = await smsService.listCards()
    if (result.ok) cards.value = result.data
    return result
  }

  async function fetchThreads(iccid?: string) {
    const result = await smsService.listContacts(iccid)
    if (result.ok) {
      threads.value = result.data
      lastOkAt.value = Date.now()
      error.value = null
    } else {
      error.value = result.error
    }
    return result
  }

  async function fetchThread(params: SmsThreadQueryParams) {
    const result = await smsService.getThread(params)
    if (result.ok) threadMessages.value = result.data
    return result
  }

  async function send(payload: SmsSendPayload) {
    return smsService.send(payload)
  }

  async function deleteMessage(id: number) {
    return smsService.deleteMessage(id)
  }

  async function markThreadRead(payload: SmsMarkThreadReadPayload) {
    return smsService.markThreadRead(payload)
  }

  async function deleteThread(payload: SmsDeleteThreadPayload) {
    return smsService.deleteThread(payload)
  }

  return {
    cards,
    threads,
    threadMessages,
    loading,
    lastOkAt,
    error,
    fetchCards,
    fetchThreads,
    fetchThread,
    send,
    markThreadRead,
    deleteMessage,
    deleteThread
  }
})
