import type { SmsCard } from '../types/sms'

export type SmsOwnIdentity = Readonly<{ iccid?: string; localPhone?: string; profileName?: string }>

export function smsIdentityLabel(identity: SmsOwnIdentity): string {
  return `${identity.localPhone || '号码未知'} · ${identity.profileName || 'SIM'} · 卡尾号 ${identity.iccid?.slice(-6) || '未知'}`
}

export type SmsCardPresentation = Readonly<{
  id: string
  label: string
  detail: string
  statusLabel: string
  online: boolean
  accessibilityLabel: string
}>

export function createSmsCardChannels(cards: readonly SmsCard[]): readonly SmsCardPresentation[] {
  return [
    { id: 'all', label: '全部卡 / profile', detail: '全部已保存的历史', statusLabel: `${cards.length} 张卡`, online: false, accessibilityLabel: '全部卡 / profile，全部已保存的历史' },
    ...cards.map(card => ({
      id: card.iccid,
      label: card.profile_name || `SIM · ${card.iccid.slice(-6)}`,
      detail: `${card.local_phone || '号码未知'} · 尾号 ${card.iccid.slice(-6)}`,
      statusLabel: card.can_send ? '当前可发送' : '仅查看历史',
      online: card.can_send,
      accessibilityLabel: `${smsIdentityLabel({ iccid: card.iccid, localPhone: card.local_phone, profileName: card.profile_name })}，${card.send_reason}`
    }))
  ]
}

export type SmsConversationContext = Readonly<{
  identityLabel: string
  deviceLabel: string
  canSend: boolean
  sendLabel: string
}>

export function createSmsConversationContext(options: Readonly<{
  thread: SmsOwnIdentity | null
  cards: readonly SmsCard[]
}>): SmsConversationContext {
  const card = options.cards.find(card => card.iccid === options.thread?.iccid)
  return {
    identityLabel: options.thread ? smsIdentityLabel({ ...options.thread, localPhone: options.thread.localPhone || card?.local_phone }) : '选择本方卡与对方号码的会话',
    deviceLabel: card?.device_name || '',
    canSend: card?.can_send === true,
    sendLabel: card?.send_reason || '卡未激活或设备离线，仍可查看历史'
  }
}

export function smsUnreadBadge(unreadCount: number, readLocally = false): number {
  return readLocally ? 0 : Math.max(0, Number(unreadCount) || 0)
}

export function normalizeSmsUnreadCount(value: unknown): number {
  return Math.max(0, Number(value) || 0)
}

export function smsPeerInitial(peer: string): string {
  return Array.from(String(peer || '').trim())[0]?.toLocaleUpperCase() || '—'
}
