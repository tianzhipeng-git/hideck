<script setup lang="ts">
import { Loading } from '@element-plus/icons-vue'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import SmsCardRail from '../components/sms/SmsCardRail.vue'
import SmsThreadListPane from '../components/sms/SmsThreadListPane.vue'
import SmsConversationHeader from '../components/sms/SmsConversationHeader.vue'
import SmsMessageTimeline from '../components/sms/SmsMessageTimeline.vue'
import SmsComposer from '../components/sms/SmsComposer.vue'
import PhoneContactsDrawer from '../components/PhoneContactsDrawer.vue'
import { Delete24Regular, Person24Regular, Send24Regular } from '@vicons/fluent'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import { useSmsWorkspace } from '../composables/useSmsWorkspace'
const { smsPageRef, cardsError, cardsLastOkAt, messagesError, messagesLastOkAt, fetchCards, refreshAll, loading, threads, cardSidebarItems, selectedCard, handleSelectCard, searchQuery, presentedThreads, selectedThreadKey, deletingThreadKey, selectThread, confirmDeleteThread, showThreadActionSheet, selectedThread, conversationContext, threadLoading, fetchMessagesAndThread, showLatestMessages, viewingTarget, messageTarget, detailScrollbar, onDetailScroll, selectedThreadGroups, canLoadMoreHistory, loadingHistoryMore, deletingMessageId, loadMoreHistory, confirmDeleteMessage, showMessageActionSheet, sendIdentityLabel, showActionSheet, isNarrowLayout, closeActionSheet, onActionSheetDelete, actionSheetTarget, showSendModal, showContactsPicker, sending, composer, composerLen, composerEstimate, sendForm, sendEstimate, selectedSendCard, selectedSendDeviceId, canSendModal, sendCardOptions, openSendModal, applyContactNumber, handleSendModal, sendToCurrentThread } = useSmsWorkspace()
</script>

<template>
  <div ref="smsPageRef" class="app-page sms-page h-[calc(100vh-144px)] flex flex-col">
    <ErrorState
      v-if="cardsError"
      class="mb-4"
      title="卡 / profile 列表加载失败"
      :message="cardsError.message"
      :status-code="cardsError.status"
      :request-method="cardsError.method"
      :request-url="cardsError.url"
      :last-success-at="cardsLastOkAt"
      retry-text="重试"
      @retry="fetchCards"
    />

    <ErrorState
      v-if="messagesError"
      class="mb-6"
      title="短信加载失败"
      :message="messagesError.message"
      :status-code="messagesError.status"
      :request-method="messagesError.method"
      :request-url="messagesError.url"
      :last-success-at="messagesLastOkAt"
      retry-text="重试"
      @retry="refreshAll"
    />

    <div class="flex-1 sms-workspace ui-card ui-workspace-glow overflow-hidden relative">
      <div v-if="loading && threads.length === 0" class="absolute inset-0 z-20 flex items-center justify-center bg-white/70 dark:bg-black/40">
        <el-icon class="is-loading" size="28"><Loading /></el-icon>
      </div>

      <div class="sms-main-layout">
        <SmsCardRail
          :items="cardSidebarItems"
          :selected-id="selectedCard"
          @select="(deviceId) => void handleSelectCard(deviceId)"
        />

        <SmsThreadListPane
          v-model="searchQuery"
          :items="presentedThreads"
          :selected-key="selectedThreadKey"
          :loading="loading"
          :deleting-key="deletingThreadKey"
          @select="(key) => void selectThread(key)"
          @delete="(thread) => void confirmDeleteThread(thread)"
          @open-actions="showThreadActionSheet"
          @new-message="openSendModal"
        />

        <section class="sms-conversation-pane flex flex-col min-w-0 min-h-0">
          <SmsConversationHeader
            :peer="selectedThread?.peer || ''"
            :context="conversationContext"
            :loading="threadLoading"
            :show-back="false"
            @refresh="() => void fetchMessagesAndThread(false)"
            @latest="() => void showLatestMessages()"
            @delete="selectedThread && void confirmDeleteThread(selectedThread)"
          />

          <div v-if="viewingTarget" class="sms-target-notice" role="status">
            <span>正在查看提醒短信</span>
            <el-button text type="primary" @click="showLatestMessages">返回最新短信</el-button>
          </div>

          <div v-if="!selectedThread" class="flex-1 flex items-center justify-center p-6">
            <EmptyState title="请选择一个会话" subtitle="从左侧联系人列表进入短信明细" />
          </div>

          <div v-else ref="detailScrollbar" class="flex-1 min-h-0 overflow-y-auto sms-detail-scroll" @scroll="onDetailScroll">
            <SmsMessageTimeline
              :groups="selectedThreadGroups"
              :loading="threadLoading"
              :can-load-more="canLoadMoreHistory"
              :loading-more="loadingHistoryMore"
              :deleting-id="deletingMessageId"
              :highlighted-id="viewingTarget ? messageTarget?.message.id : undefined"
              @load-more="loadMoreHistory"
              @delete="(message) => void confirmDeleteMessage(message)"
              @open-actions="showMessageActionSheet"
            />
          </div>

          <SmsComposer
            v-if="selectedThread"
            v-model="composer"
            :encoding="composerEstimate.encoding"
            :parts="composerEstimate.parts"
            :length="composerLen"
            :sending="sending"
            :can-send="conversationContext.canSend"
            :send-label="sendIdentityLabel"
            @send="sendToCurrentThread"
          />
        </section>
      </div>
    </div>

    <transition name="sms-sheet-fade">
      <div v-if="showActionSheet && isNarrowLayout" class="sms-action-sheet-mask" @click="closeActionSheet">
        <div class="sms-action-sheet" @click.stop>
          <div class="sms-action-sheet-title">操作</div>
          <el-button class="sms-danger-ghost-btn !w-full !justify-center" @click="void onActionSheetDelete()">
            <el-icon><Delete24Regular /></el-icon>
            {{ actionSheetTarget?.type === 'thread' ? '删除对话' : '删除短信' }}
          </el-button>
          <el-button class="!w-full !justify-center" @click="closeActionSheet">取消</el-button>
        </div>
      </div>
    </transition>

    <!-- Send Modal -->
    <el-dialog v-model="showSendModal" title="发送短信" width="min(520px, 92vw)" class="glass-modal">
      <el-form label-position="top" class="mt-2">
        <el-form-item label="本方卡 / profile">
          <el-select v-model="selectedSendCard" placeholder="选择发送使用的卡">
            <el-option v-for="opt in sendCardOptions" :key="opt.value" :label="opt.label" :value="opt.value" :disabled="opt.disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标号码">
          <div class="sms-recipient-field">
            <el-input v-model="sendForm.phone" placeholder="+86138..." />
            <el-button class="sms-contact-picker-button" aria-label="从联系人选择" @click="showContactsPicker = true">
              <el-icon><Person24Regular /></el-icon>
              联系人
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="短信内容">
          <el-input
            v-model="sendForm.message"
            type="textarea"
            placeholder="输入短信内容..."
            :autosize="{ minRows: 4, maxRows: 10 }"
            resize="none"
          />
          <div class="mt-2 text-xs flex justify-end text-[var(--ui-muted)]">
            {{ sendEstimate.encoding }} · 预计 {{ sendEstimate.parts }} 段 · {{ Array.from(String(sendForm.message || '')).length }} 字
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <el-button @click="showSendModal = false">取消</el-button>
          <el-button type="primary" :loading="sending" :disabled="!canSendModal" @click="handleSendModal">
            <el-icon><Send24Regular /></el-icon>
            发送
          </el-button>
        </div>
      </template>
    </el-dialog>

    <PhoneContactsDrawer
      v-model="showContactsPicker"
      :device-id="selectedSendDeviceId"
      kind="sms"
      @select="applyContactNumber"
    />
  </div>
</template>

<style scoped>
.sms-target-notice { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 4px 16px; border-bottom: 1px solid var(--ui-border); background: var(--ui-selected); color: var(--ui-text); font-size: 13px; }
.sms-target-notice :deep(.el-button) { min-height: 44px; flex-shrink: 0; }
.sms-page {
  container-type: inline-size;
}

.sms-recipient-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  width: 100%;
}

.sms-recipient-field :deep(.el-input) {
  min-width: 0;
}

.sms-contact-picker-button {
  min-width: 88px;
  margin: 0;
}

.sms-workspace {
  min-height: 0;
  animation: sms-workspace-enter 220ms var(--ui-ease-out) both;
}

.sms-main-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0;
  height: 100%;
  min-height: 0;
}

.sms-conversation-pane {
  overflow: hidden;
  background: transparent;
}

.sms-workspace :deep(.sms-device-rail),
.sms-workspace :deep(.sms-thread-list),
.sms-workspace :deep(.sms-conversation-header) {
  background: transparent;
}

.sms-workspace :deep(.sms-composer) {
  background: color-mix(in srgb, var(--ui-surface) 82%, transparent);
}

@keyframes sms-workspace-enter {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

@media (prefers-reduced-motion: reduce) {
  .sms-workspace { animation-name: sms-workspace-fade; }
  @keyframes sms-workspace-fade { from { opacity: 0; } to { opacity: 1; } }
}

.sms-action-sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 2200;
  background: color-mix(in srgb, var(--ui-primary-solid) 36%, transparent);
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.sms-action-sheet {
  width: min(520px, 100%);
  background: var(--ui-surface);
  border-top-left-radius: 8px;
  border-top-right-radius: 8px;
  border: 1px solid var(--ui-border);
  border-bottom: none;
  box-shadow: var(--ui-shadow-lg);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sms-action-sheet-title {
  font-size: 12px;
  font-weight: 700;
  color: var(--ui-text-muted);
  text-align: center;
  letter-spacing: 0.03em;
}

.sms-sheet-fade-enter-active,
.sms-sheet-fade-leave-active {
  transition: opacity 0.18s ease;
}

.sms-sheet-fade-enter-from,
.sms-sheet-fade-leave-to {
  opacity: 0;
}

:deep(.sms-danger-ghost-btn.el-button) {
  color: var(--ui-danger);
  border-color: color-mix(in srgb, var(--ui-danger) 24%, var(--ui-border));
  background: color-mix(in srgb, var(--ui-danger) 9%, transparent);
}

:deep(.sms-danger-ghost-btn.el-button:hover) {
  color: var(--ui-danger);
  border-color: color-mix(in srgb, var(--ui-danger) 38%, var(--ui-border));
  background: color-mix(in srgb, var(--ui-danger) 15%, transparent);
}

:deep(.sms-danger-ghost-btn.el-button:focus-visible) {
  outline: 2px solid color-mix(in srgb, var(--ui-danger) 25%, transparent);
  outline-offset: 2px;
}

@container (min-width: 980px) {
  .sms-main-layout {
    grid-template-columns: 218px 310px minmax(0, 1fr);
  }
}

@container (max-width: 979px) {
  .sms-workspace {
    flex: none;
    min-height: 870px;
    overflow: hidden;
  }

  .sms-main-layout {
    min-height: 870px;
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: auto 280px minmax(620px, 1fr);
  }

  .sms-main-layout > :deep(.sms-device-rail) {
    grid-column: 1;
    grid-row: 1;
  }

  .sms-main-layout > :deep(.sms-thread-list) {
    grid-column: 1;
    grid-row: 2;
    border-right: 0;
    border-bottom: 1px solid var(--ui-border);
  }

  .sms-conversation-pane {
    min-height: 620px;
    grid-column: 1;
    grid-row: 3;
  }
}

@media (max-width: 1180px) {
  .sms-page {
    height: auto !important;
    min-height: calc(100dvh - 104px);
  }
}

@container (max-width: 560px) {
  .sms-main-layout {
    grid-template-columns: minmax(0, 1fr);
  }

}

/* 消息详情区域原生滚动条样式 */
.sms-detail-scroll::-webkit-scrollbar {
  width: 6px;
}
.sms-detail-scroll::-webkit-scrollbar-track {
  background: transparent;
}
.sms-detail-scroll::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--ui-text-muted) 30%, transparent);
  border-radius: 3px;
}
.sms-detail-scroll::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--ui-text-muted) 50%, transparent);
}
/* Firefox */
.sms-detail-scroll {
  scrollbar-width: thin;
  scrollbar-color: color-mix(in srgb, var(--ui-text-muted) 30%, transparent) transparent;
}
</style>
