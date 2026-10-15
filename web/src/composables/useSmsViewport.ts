import { computed, onMounted, onUnmounted, ref } from 'vue'

export function useSmsViewport() {
  const smsPageRef = ref<HTMLElement | null>(null)
  const width = ref(0)
  const isNarrowLayout = computed(() => width.value > 0 && width.value < 980)
  let observer: ResizeObserver | null = null
  const sync = () => { width.value = smsPageRef.value?.clientWidth || 0 }
  onMounted(() => {
    sync()
    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(sync)
      if (smsPageRef.value) observer.observe(smsPageRef.value)
    } else {
      window.addEventListener('resize', sync, { passive: true })
    }
  })
  onUnmounted(() => {
    observer?.disconnect()
    window.removeEventListener('resize', sync)
  })
  return { smsPageRef, isNarrowLayout }
}
