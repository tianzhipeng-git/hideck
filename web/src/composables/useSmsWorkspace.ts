import { useSmsRouteSync } from './useSmsRouteSync'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useSmsViewport } from './useSmsViewport'
import { useSmsSending } from '../composables/useSmsSending'
import { useSMSStore } from '../stores/sms'
import { usePollingScheduler } from '../composables/usePollingScheduler'
import { toAppError } from '../services/http'
import { resolveSmsThreadKey, type SmsThreadQueryParams } from '../services/sms'
import { fetchSmsMessageTarget, type SmsMessageTarget } from '../services/sms-message-target'
import { includeSmsTargetThread } from '../utils/smsMessageTarget'
import type { SMSMessage } from '../types/api'
import type { SmsCard } from '../types/sms'
import type { SmsThreadVM as SmsThread } from '../types/view-model'
import { formatDeviceDate } from '../utils/deviceTime'
import { smsIdentityLabel, createSmsConversationContext, createSmsCardChannels } from '../utils/smsPresentation'
import { isNearTimelineEnd } from '../utils/timelineFollow'
import {
  SMS_THREAD_PAGE_SIZE,
  mergeSmsThreadPages,
  threadAlreadyHasLatest
} from '../utils/smsThreadMessages'

export function useSmsWorkspace() {
  const route = useRoute()
  const router = useRouter()
  const smsStore = useSMSStore()

  const cards = ref<SmsCard[]>([])
  const cardsLastOkAt = ref<number | null>(null)
  const cardsError = ref<{ message: string; status?: number; method?: string; url?: string } | null>(null)

  const loading = ref(false)
  const threads = ref<SmsThread[]>([])
  const messagesLastOkAt = ref<number | null>(null)
  const messagesError = ref<{ message: string; status?: number; method?: string; url?: string } | null>(null)

  const selectedCard = ref<string>(typeof route.query.card === 'string' ? route.query.card : 'all')
  const selectedThreadKey = ref<string>(typeof route.query.contact === 'string' ? route.query.contact : '')
  const targetMessageQuery = ref<unknown>(route.query.message)
  const messageTarget = ref<SmsMessageTarget | null>(null)
  const viewingTarget = computed(() => !!messageTarget.value && route.query.latest !== '1')
  const searchQuery = ref('')

  const { smsPageRef, isNarrowLayout } = useSmsViewport()

  function dateKey(timestamp: string) {
    return formatDeviceDate(timestamp, { fallback: '未知日期' })
  }

  const filteredThreads = computed(() => {
    const q = String(searchQuery.value || '').trim().toLowerCase()
    if (!q) return threads.value
    return threads.value.filter(t => {
      if (t.peerLower.includes(q)) return true
      return t.lastMessageLower.includes(q) || (t.localPhone || '').includes(q) || (t.profileName || '').toLowerCase().includes(q) || t.iccid.includes(q)
    })
  })

  const presentedThreads = computed(() => {
    return filteredThreads.value
  })

  const selectedThread = computed(() => {
    return threads.value.find(t => t.key === selectedThreadKey.value) || null
  })

  const loadingHistoryMore = ref(false)
  const threadLoading = ref(false)
  const threadMessages = ref<SMSMessage[]>([])
  const threadHasMore = ref(false)

  const canLoadMoreHistory = computed(() => {
    return !!selectedThread.value && threadHasMore.value
  })

  const selectedThreadGroups = computed(() => {
    if (!selectedThread.value) return []
    const out: Array<{ date: string; items: SMSMessage[] }> = []
    let last = ''
    for (const m of threadMessages.value) {
      const key = dateKey(m.timestamp)
      if (!out.length || key !== last) {
        out.push({ date: key, items: [m] })
        last = key
      } else {
        out[out.length - 1].items.push(m)
      }
    }
    return out
  })


  const deletingMessageId = ref<number | null>(null)
  const deletingThreadKey = ref<string | null>(null)
  const showActionSheet = ref(false)
  const actionSheetTarget = ref<{ type: 'thread'; thread: SmsThread } | { type: 'message'; message: SMSMessage } | null>(null)
  const detailScrollbar = ref<HTMLElement | null>(null)
  const followingLatest = ref(true)
  let followLatestTimer = 0

  const { showSendModal, showContactsPicker, sending, composer, composerLen, composerEstimate, sendForm, sendEstimate, selectedSendCard, selectedSendDeviceId, canSendModal, sendCardOptions, openSendModal, applyContactNumber, handleSendModal, sendToCurrentThread } = useSmsSending({ cards, selectedCard, selectedThread, selectedThreadKey, viewingTarget, showLatestMessages, scrollThreadToBottom, fetchMessagesAndThread })

  const sendIdentityLabel = computed(() => {
    const card = cards.value.find(card => card.iccid === selectedThread.value?.iccid)
    if (!card?.can_send) return conversationContext.value.sendLabel
    return `发送使用：${smsIdentityLabel({ iccid: card.iccid, localPhone: card.local_phone, profileName: card.profile_name })} · 当前设备：${card.device_name}`
  })
  const cardSidebarItems = computed(() => createSmsCardChannels(cards.value))
  const conversationContext = computed(() => createSmsConversationContext({
    thread: selectedThread.value,
    cards: cards.value
  }))

  function normalizeQueryCard(card: string) {
    const v = String(card || '').trim()
    if (!v || v === 'all') return undefined
    return v
  }

  function buildSmsQuery(card: string, contact?: string) {
    const nextContact = String(contact || '').trim()
    return {
      ...route.query,
      message: undefined,
      latest: undefined,
      device: undefined,
      card: normalizeQueryCard(card),
      contact: nextContact || undefined
    }
  }

  async function markThreadSeen(t: SmsThread | null) {
    if (!t || threadMessages.value.length === 0) return true
    const hasDisplayedUnread = threadMessages.value.some(message => message.type === 1 && message.status === 0)
    if (t.unreadCount <= 0 && !hasDisplayedUnread) return true
    const messageIDs = threadMessages.value.map(message => message.id)
    const scope = viewingTarget.value
      ? { message_ids: messageIDs }
      : { through_id: Math.max(...messageIDs) }
    if (!t.iccid) {
      messagesError.value = { message: '短信会话缺少 ICCID，无法保存已读状态' }
      return false
    }
    const result = await smsStore.markThreadRead({ iccid: t.iccid, peer: t.peer, ...scope })
    if (!result.ok) {
      messagesError.value = result.error
      return false
    }
    threads.value = threads.value.map(thread => (
      thread.key === t.key ? { ...thread, unreadCount: result.data.unread_count } : thread
    ))
    return true
  }

  function scrollThreadToBottom() {
    followingLatest.value = true
    const snap = () => {
      const wrap = detailScrollbar.value
      if (!wrap) return
      wrap.scrollTop = wrap.scrollHeight
    }
    nextTick(() => {
      snap()
      requestAnimationFrame(() => {
        snap()
        requestAnimationFrame(snap)
      })
      window.clearTimeout(followLatestTimer)
      followLatestTimer = window.setTimeout(snap, 80)
    })
  }

  function getDetailWrap() {
    return detailScrollbar.value
  }

  async function loadMoreHistory() {
    if (!canLoadMoreHistory.value) return
    if (loadingHistoryMore.value) return
    if (!selectedThread.value) return
    if (threadMessages.value.length === 0) return
    const wrap = getDetailWrap()
    const prevTop = wrap?.scrollTop || 0
    const prevHeight = wrap?.scrollHeight || 0
    const requestThreadKey = selectedThreadKey.value
    loadingHistoryMore.value = true
    try {
      const oldest = threadMessages.value[0]
      const params: SmsThreadQueryParams = {
        peer: selectedThread.value.peer,
        limit: SMS_THREAD_PAGE_SIZE,
        before_ts: oldest.timestamp,
        before_id: oldest.id
      }
      params.iccid = selectedThread.value.iccid
      const result = await smsStore.fetchThread(params)
      if (selectedThreadKey.value !== requestThreadKey) return
      if (!result.ok) {
        messagesError.value = result.error
        return
      }
      const list = (result.data || []) as SMSMessage[]
      threadMessages.value = mergeSmsThreadPages(list, threadMessages.value)
      threadHasMore.value = list.length === params.limit
      await nextTick()
      requestAnimationFrame(() => {
        const w = getDetailWrap()
        if (!w) return
        const nextHeight = w.scrollHeight
        const delta = Math.max(0, nextHeight - prevHeight)
        w.scrollTop = prevTop + delta
      })
      messagesError.value = null
      if (viewingTarget.value) await markThreadSeen(selectedThread.value)
    } catch (error: unknown) {
      if (selectedThreadKey.value !== requestThreadKey) return
      messagesError.value = toAppError(error)
    } finally {
      loadingHistoryMore.value = false
    }
  }

  function onDetailScroll(e: Event) {
    const target = e.target as HTMLElement
    if (!target) return
    followingLatest.value = isNearTimelineEnd({
      scrollTop: target.scrollTop,
      clientHeight: target.clientHeight,
      scrollHeight: target.scrollHeight
    }, 160)
    if (target.scrollTop <= 80) {
      loadMoreHistory()
    }
  }

  let cardsFetchSeq = 0
  let messagesFetchSeq = 0
  let threadFetchSeq = 0
  function showThreadActionSheet(thread: SmsThread) {
    actionSheetTarget.value = { type: 'thread', thread }
    showActionSheet.value = true
  }

  function showMessageActionSheet(message: SMSMessage) {
    actionSheetTarget.value = { type: 'message', message }
    showActionSheet.value = true
  }

  function closeActionSheet() {
    showActionSheet.value = false
    actionSheetTarget.value = null
  }

  async function onActionSheetDelete() {
    const target = actionSheetTarget.value
    closeActionSheet()
    if (!target) return
    if (target.type === 'thread') {
      await confirmDeleteThread(target.thread)
      return
    }
    await confirmDeleteMessage(target.message)
  }

  function clearSelectedThread(syncRoute = false) {
    targetMessageQuery.value = undefined
    messageTarget.value = null
    messagesFetchSeq += 1
    threadFetchSeq += 1
    selectedThreadKey.value = ''
    threadMessages.value = []
    threadHasMore.value = false
    threadLoading.value = false
    if (syncRoute) {
      void router.replace({ query: buildSmsQuery(selectedCard.value) })
    }
  }

  async function fetchCards() {
    const seq = ++cardsFetchSeq
    cardsError.value = null
    const result = await smsStore.fetchCards()
    if (seq !== cardsFetchSeq) return false
    if (result.ok) {
      cards.value = (result.data || []) as SmsCard[]
      cardsLastOkAt.value = Date.now()
      return true
    }
    cards.value = []
    cardsError.value = result.error
    return false
  }

  async function fetchMessages(silent = false) {
    const seq = ++messagesFetchSeq
    if (!silent) loading.value = true
    messagesError.value = null
    const result = await smsStore.fetchThreads(selectedCard.value)
    if (seq !== messagesFetchSeq) return false
    if (targetMessageQuery.value !== undefined) {
      const target = await fetchSmsMessageTarget(targetMessageQuery.value)
      if (seq !== messagesFetchSeq) return false
      if (!target.ok) {
        messagesError.value = target.error
        loading.value = false
        return false
      }
      messageTarget.value = target.data
      selectedThreadKey.value = target.data.thread.key
    }
    if (result.ok) {
      threads.value = messageTarget.value ? includeSmsTargetThread(result.data, messageTarget.value.thread) : result.data
      messagesLastOkAt.value = Date.now()
    } else {
      messagesError.value = result.error
    }
    if (!silent) loading.value = false
    return result.ok
  }

  async function fetchThreadLatest(silent = false) {
    const t = selectedThread.value
    if (!t) {
      threadMessages.value = []
      threadHasMore.value = false
      return false
    }
    if (viewingTarget.value && messageTarget.value?.thread.key === t.key) {
      threadMessages.value = messageTarget.value.messages
      threadHasMore.value = messageTarget.value.hasMore
      return true
    }
    if (silent && threadAlreadyHasLatest(threadMessages.value, t.lastSmsId)) {
      return true
    }
    const seq = ++threadFetchSeq
    if (!silent) threadLoading.value = true
    const params: SmsThreadQueryParams = { peer: t.peer, limit: SMS_THREAD_PAGE_SIZE }
    params.iccid = t.iccid
    const result = await smsStore.fetchThread(params)
    if (seq !== threadFetchSeq) return false
    if (result.ok) {
      const list = (result.data || []) as SMSMessage[]
      if (silent && threadMessages.value.length > 0) {
        threadMessages.value = mergeSmsThreadPages(threadMessages.value, list)
        if (!threadHasMore.value) threadHasMore.value = list.length === params.limit
      } else {
        threadMessages.value = list
        threadHasMore.value = list.length === params.limit
      }
    } else {
      messagesError.value = result.error
    }
    if (!silent) threadLoading.value = false
    return result.ok
  }

  async function ensureThreadSelection(options: { syncRoute?: boolean; silent?: boolean; scrollToBottom?: boolean } = {}) {
    const syncRoute = options.syncRoute === true
    const silent = options.silent === true
    const scrollToBottom = options.scrollToBottom ?? !viewingTarget.value

    const requestedKey = selectedThreadKey.value
    let current = selectedThread.value
    if (!current && requestedKey) {
      const resolvedKey = resolveSmsThreadKey(requestedKey, threads.value)
      if (resolvedKey) {
        selectedThreadKey.value = resolvedKey
        current = selectedThread.value
        if (resolvedKey !== requestedKey) {
          void router.replace({ query: buildSmsQuery(selectedCard.value, resolvedKey) })
        }
      }
    }
    if (current) {
      const previousCount = threadMessages.value.length
      const previousLastId = threadMessages.value.at(-1)?.id
      const ok = await fetchThreadLatest(silent)
      if (ok) {
        await applyThreadSeen(current)
        const changed = threadMessages.value.length !== previousCount || threadMessages.value.at(-1)?.id !== previousLastId
        if (!viewingTarget.value && (scrollToBottom || (silent && followingLatest.value && changed))) {
          scrollThreadToBottom()
        }
      }
      return
    }

    threadMessages.value = []
    threadHasMore.value = false
    if (requestedKey) {
      selectedThreadKey.value = ''
      void router.replace({ query: buildSmsQuery(selectedCard.value) })
      return
    }
    if (isNarrowLayout.value || filteredThreads.value.length === 0) return
    await selectThread(filteredThreads.value[0].key, { syncRoute, silent, scrollToBottom })
  }

  async function selectThread(key: string, options: { syncRoute?: boolean; silent?: boolean; scrollToBottom?: boolean } = {}) {
    if (!key) return
    if (selectedThreadKey.value === key && threadMessages.value.length > 0) return
    targetMessageQuery.value = undefined
    messageTarget.value = null
    messagesFetchSeq += 1

    const syncRoute = options.syncRoute !== false
    const silent = options.silent === true
    const scrollToBottom = options.scrollToBottom !== false
    followingLatest.value = scrollToBottom

    selectedThreadKey.value = key
    threadFetchSeq += 1
    threadMessages.value = []
    threadHasMore.value = false
    threadLoading.value = false
    messagesError.value = null
    if (syncRoute) {
      void router.replace({ query: buildSmsQuery(selectedCard.value, key) })
    }

    const t = threads.value.find(x => x.key === key) || null
    if (!t) {
      threadMessages.value = []
      threadHasMore.value = false
      return
    }

    const ok = await fetchThreadLatest(silent)
    if (!ok) return
    await applyThreadSeen(t)
    if (scrollToBottom) scrollThreadToBottom()
  }

  async function applyThreadSeen(thread: SmsThread) {
    await markThreadSeen(thread)
  }

  async function handleSelectCard(iccid: string, options: { syncRoute?: boolean; silent?: boolean } = {}) {
    const nextCard = String(iccid || 'all').trim() || 'all'
    const syncRoute = options.syncRoute !== false
    const silent = options.silent === true

    selectedCard.value = nextCard
    clearSelectedThread(false)
    if (syncRoute) {
      void router.replace({ query: buildSmsQuery(nextCard) })
    }

    const ok = await fetchMessages(silent)
    if (!ok || selectedCard.value !== nextCard) return
    await ensureThreadSelection({ syncRoute, silent, scrollToBottom: !viewingTarget.value })
  }

  async function fetchMessagesAndThread(silent = false) {
    const ok = await fetchMessages(silent)
    if (!ok) return
    await ensureThreadSelection({ syncRoute: false, silent, scrollToBottom: !silent })
  }

  async function refreshAll() {
    await fetchCards()
    await fetchMessagesAndThread()
  }

  async function showLatestMessages() {
    followingLatest.value = true
    if (viewingTarget.value) {
      targetMessageQuery.value = undefined
      messageTarget.value = null
      void router.replace({ query: { ...route.query, latest: '1' } })
      const ok = await fetchThreadLatest(false)
      if (ok) scrollThreadToBottom()
      return
    }
    scrollThreadToBottom()
  }

  async function pollRefresh() {
    if (loading.value || threadLoading.value || loadingHistoryMore.value) return
    try {
      await Promise.all([fetchCards(), fetchMessagesAndThread(true)])
    } catch {
      // Ignore polling failures; scheduler keeps retrying.
    }
  }

  usePollingScheduler(pollRefresh, 5000, {
    maxIntervalMs: 60000,
    backgroundIntervalMs: 15000
  })

  watch(
    () => isNarrowLayout.value,
    async (isNarrow) => {
      if (isNarrow) return
      if (selectedThreadKey.value) return
      if (filteredThreads.value.length === 0) return
      await selectThread(filteredThreads.value[0].key, { syncRoute: true, scrollToBottom: true })
    }
  )

  useSmsRouteSync({ route, selectedCard, selectedThreadKey, targetMessageQuery, clearSelectedThread, fetchMessagesAndThread })

  onMounted(async () => {
    const [, messagesOk] = await Promise.all([fetchCards(), fetchMessages()])
    if (messagesOk) await ensureThreadSelection({ syncRoute: false, silent: false, scrollToBottom: !viewingTarget.value })
  })

  onUnmounted(() => {
    messagesFetchSeq += 1
    cardsFetchSeq += 1
    threadFetchSeq += 1
    window.clearTimeout(followLatestTimer)
  })

  async function confirmDeleteMessage(message: SMSMessage) {
    if (!message.id || deletingMessageId.value === message.id) return
    try {
      await ElMessageBox.confirm(
        '删除后无法恢复。仅删除短信中心历史记录。',
        '删除这条短信？',
        {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return
    }

    deletingMessageId.value = message.id
    try {
      const result = await smsStore.deleteMessage(message.id)
      if (!result.ok) throw new Error(result.error.message || '删除失败')
      ElMessage.success('已删除短信')
      if (messageTarget.value?.message.id === message.id) {
        clearSelectedThread(true)
        await fetchMessages(false)
        return
      }
      await fetchMessagesAndThread()
      if (result.data.thread_empty) {
        clearSelectedThread(true)
      }
    } catch (e: unknown) {
      const err = toAppError(e)
      ElMessage.error('删除失败：' + (err.message || '未知错误'))
    } finally {
      deletingMessageId.value = null
    }
  }

  async function confirmDeleteThread(thread: SmsThread) {
    if (deletingThreadKey.value === thread.key) return
    if (!thread.iccid) {
      ElMessage.error('短信会话缺少 ICCID，无法确定要删除的会话')
      return
    }
    try {
      await ElMessageBox.confirm(
        `将删除与 ${thread.peer} 的全部短信历史，无法恢复。仅删除短信中心历史记录。`,
        '永久删除整个对话？',
        {
          confirmButtonText: '删除对话',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return
    }

    deletingThreadKey.value = thread.key
    try {
      const payload = { iccid: thread.iccid, peer: thread.peer }
      const result = await smsStore.deleteThread(payload)
      if (!result.ok) throw new Error(result.error.message || '删除失败')
      ElMessage.success('已删除对话')
      const deletingCurrent = selectedThreadKey.value === thread.key
      if (deletingCurrent) clearSelectedThread(true)
      await fetchMessages(false)
    } catch (e: unknown) {
      const err = toAppError(e)
      ElMessage.error('删除失败：' + (err.message || '未知错误'))
    } finally {
      deletingThreadKey.value = null
    }
  }


  return { smsPageRef, cardsError, cardsLastOkAt, messagesError, messagesLastOkAt, fetchCards, refreshAll, loading, threads, cardSidebarItems, selectedCard, handleSelectCard, searchQuery, presentedThreads, selectedThreadKey, deletingThreadKey, selectThread, confirmDeleteThread, showThreadActionSheet, selectedThread, conversationContext, threadLoading, fetchMessagesAndThread, showLatestMessages, viewingTarget, messageTarget, detailScrollbar, onDetailScroll, selectedThreadGroups, canLoadMoreHistory, loadingHistoryMore, deletingMessageId, loadMoreHistory, confirmDeleteMessage, showMessageActionSheet, sendIdentityLabel, showActionSheet, isNarrowLayout, closeActionSheet, onActionSheetDelete, actionSheetTarget, showSendModal, showContactsPicker, sending, composer, composerLen, composerEstimate, sendForm, sendEstimate, selectedSendCard, selectedSendDeviceId, canSendModal, sendCardOptions, openSendModal, applyContactNumber, handleSendModal, sendToCurrentThread }

}
