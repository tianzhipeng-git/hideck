<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { cardsService } from '../services/cards'

const props = defineProps<{ iccid: string; modelValue: string; ready: boolean; showSensitive: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: string]; retry: [] }>()
const editing = ref(false)
const draft = ref('')
const saving = ref(false)
function edit() {
  draft.value = props.modelValue
  editing.value = true
}
async function save() {
  if (saving.value) return
  saving.value = true
  const result = await cardsService.putPhoneNote(props.iccid, draft.value.trim())
  saving.value = false
  if (!result.ok) { ElMessage.error(result.error.message); return }
  emit('update:modelValue', result.data.manual_phone_number)
  editing.value = false
  ElMessage.success(result.data.manual_phone_number ? '备注号码已保存' : '备注号码已清除')
}
</script>

<template>
  <div class="mt-1 ml-4 text-xs text-[var(--ui-muted)]">
    <template v-if="!ready">
      <el-button size="small" link @click="emit('retry')">重新加载备注号码</el-button>
    </template>
    <template v-else-if="!editing">
      备注号码：<span :class="{ 'blur-sm select-none': !showSensitive }">{{ modelValue || '未填写' }}</span>
      <el-button size="small" link type="primary" @click="edit">编辑</el-button>
    </template>
    <template v-else>
      <div class="flex flex-wrap items-center gap-2">
        <el-input v-model="draft" aria-label="备注手机号码" placeholder="手机号，可带 + 国际区号" size="small" class="!w-52" :disabled="saving" @keyup.enter="save" @keyup.escape="editing = false" />
        <el-button size="small" type="primary" :loading="saving" @click="save">保存</el-button>
        <el-button size="small" :disabled="saving" @click="editing = false">取消</el-button>
      </div>
      <div class="mt-1">留空并保存可清除；自动获取不到号码时，短信中心使用此备注。</div>
    </template>
  </div>
</template>
