import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { cardsService } from '../services/cards'

export function useProfilePhoneNotes() {
  const phoneNotes = ref<Record<string, string>>({})
  const phoneNotesReady = ref(false)
  async function loadPhoneNotes() {
    const result = await cardsService.getPhoneNotes()
    if (result.ok) {
      phoneNotes.value = result.data
      phoneNotesReady.value = true
    } else {
      ElMessage.error(result.error.message)
    }
  }
  onMounted(loadPhoneNotes)
  const phoneNoteKey = (iccid: string) => iccid.toUpperCase().replace(/\s/g, '').replace(/F+$/, '')
  return { phoneNotes, phoneNotesReady, loadPhoneNotes, phoneNoteKey }
}
