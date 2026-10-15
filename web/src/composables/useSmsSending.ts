import { computed, ref, type Ref, type ComputedRef } from 'vue'
import { ElMessage } from 'element-plus'
import { useSMSStore } from '../stores/sms'
import { toAppError } from '../services/http'
import { smsIdentityLabel } from '../utils/smsPresentation'
import type { SmsCard } from '../types/sms'
import type { SmsThreadVM } from '../types/view-model'

export function useSmsSending(options: {
  cards: Ref<SmsCard[]>; selectedCard: Ref<string>; selectedThread: ComputedRef<SmsThreadVM | null>
  selectedThreadKey: Ref<string>; viewingTarget: ComputedRef<boolean>
  showLatestMessages: () => Promise<void>; scrollThreadToBottom: () => void
  fetchMessagesAndThread: (silent?: boolean) => Promise<void>
}) {
  const { cards, selectedCard, selectedThread, selectedThreadKey, viewingTarget, showLatestMessages, scrollThreadToBottom, fetchMessagesAndThread } = options
  const smsStore = useSMSStore()
  const showSendModal = ref(false)
  const showContactsPicker = ref(false)
  const sending = ref(false)
  const composer = ref('')
  const GSM7_BASIC_CHARS = new Set(Array.from(`@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞ !"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà`))
  const GSM7_EXT_CHARS = new Set(Array.from(`^{}\\[~]|€`))

  function estimateSegments(text: string) {
    const raw = String(text || '')
    let gsm7Units = 0
    let isGSM7 = true
    for (const ch of Array.from(raw)) {
      if (GSM7_BASIC_CHARS.has(ch)) {
        gsm7Units += 1
        continue
      }
      if (GSM7_EXT_CHARS.has(ch)) {
        gsm7Units += 2
        continue
      }
      isGSM7 = false
      break
    }
    if (isGSM7) {
      const single = 160
      const multi = 153
      const parts = gsm7Units <= single ? 1 : Math.ceil(gsm7Units / multi)
      return { encoding: 'GSM7', parts, units: gsm7Units, unitName: 'septets' }
    }
    const ucs2Units = Array.from(raw).length
    const single = 70
    const multi = 67
    const parts = ucs2Units <= single ? 1 : Math.ceil(ucs2Units / multi)
    return { encoding: 'UCS2', parts, units: ucs2Units, unitName: 'chars' }
  }

  const composerLen = computed(() => Array.from(String(composer.value || '')).length)
  const composerEstimate = computed(() => estimateSegments(String(composer.value || '')))
  const sendForm = ref({
    phone: '',
    message: ''
  })
  const sendEstimate = computed(() => estimateSegments(String(sendForm.value.message || '')))

  const selectedSendCard = ref('')
  const selectedSendDeviceId = computed(() => cards.value.find(card => card.iccid === selectedSendCard.value)?.device_id || '')
  const canSendModal = computed(() => cards.value.some(card => card.iccid === selectedSendCard.value && card.can_send))
  const sendCardOptions = computed(() => cards.value.map(card => ({
    label: smsIdentityLabel({ iccid: card.iccid, localPhone: card.local_phone, profileName: card.profile_name }),
    value: card.iccid, disabled: !card.can_send
  })))

  function openSendModal() {
    sendForm.value.phone = ''
    sendForm.value.message = ''
    selectedSendCard.value = selectedCard.value !== 'all' ? selectedCard.value : (cards.value.find(card => card.can_send)?.iccid || '')
    showSendModal.value = true
  }

  function applyContactNumber(number: string) {
    sendForm.value.phone = number
    showContactsPicker.value = false
  }

  async function handleSendModal() {
    if (!canSendModal.value || !sendForm.value.phone || !sendForm.value.message) {
      ElMessage.warning('请填写完整信息')
      return
    }
    sending.value = true
    try {
      const result = await smsStore.send({
        iccid: selectedSendCard.value,
        phone: sendForm.value.phone,
        message: sendForm.value.message
      })
      if (!result.ok) throw new Error(result.error.message || '发送失败')
      const parts = result.data.partsTotal
      ElMessage.success(`短信已发送${parts > 1 ? `（${parts}段）` : ''}`)
      showSendModal.value = false
      setTimeout(async () => {
        await fetchMessagesAndThread()
      }, 800)
    } catch (e: unknown) {
      const err = toAppError(e)
      ElMessage.error('发送失败：' + (err.message || '未知错误'))
    } finally {
      sending.value = false
    }
  }

  async function sendToCurrentThread() {
    const t = selectedThread.value
    if (!t) return
    if (!cards.value.some(card => card.iccid === t.iccid && card.can_send)) {
      ElMessage.warning('此卡当前不可发送，历史记录仍可查看')
      return
    }
    const text = String(composer.value || '').trim()
    if (!text) return
    if (!t.iccid) {
      ElMessage.error('短信会话缺少 ICCID，无法确定外呼 SIM 卡')
      return
    }
    sending.value = true
    try {
      const result = await smsStore.send({ iccid: t.iccid, phone: t.peer, message: text })
      if (!result.ok) throw new Error(result.error.message || '发送失败')
      composer.value = ''
      if (viewingTarget.value && selectedThreadKey.value === t.key) {
        showLatestMessages()
        return
      }
      scrollThreadToBottom()
      setTimeout(async () => {
        await fetchMessagesAndThread()
      }, 800)
    } catch (e: unknown) {
      const err = toAppError(e)
      ElMessage.error('发送失败：' + (err.message || '未知错误'))
    } finally {
      sending.value = false
    }
  }


  return { showSendModal, showContactsPicker, sending, composer, composerLen, composerEstimate, sendForm, sendEstimate, selectedSendCard, selectedSendDeviceId, canSendModal, sendCardOptions, openSendModal, applyContactNumber, handleSendModal, sendToCurrentThread }

}
