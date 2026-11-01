import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { mapsApi } from '../api/maps'

export const useMapStore = defineStore('map', () => {
  const progress = ref(null)
  const loading = ref(false)

  const completedLevels = computed(() => progress.value?.completedLevels || 0)
  const totalLevels = computed(() => progress.value?.totalLevels || 42)
  const totalXp = computed(() => progress.value?.totalXp ?? 0)
  const currentMap = computed(() => progress.value?.currentMapName || '')

  const overallPct = computed(() =>
    totalLevels.value ? Math.round((completedLevels.value / totalLevels.value) * 100) : 0
  )

  async function fetchProgress(force = false) {
    if (loading.value) return progress.value
    if (progress.value && !force) return progress.value
    loading.value = true
    try {
      const res = await mapsApi.progress()
      progress.value = res.data.data
    } catch (e) {
      console.error(e)
    } finally {
      loading.value = false
    }
    return progress.value
  }

  function refresh() {
    progress.value = null
    return fetchProgress(true)
  }

  return { progress, loading, completedLevels, totalLevels, totalXp, currentMap, overallPct, fetchProgress, refresh }
})
