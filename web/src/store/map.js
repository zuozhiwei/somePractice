import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { mapsApi } from '../api/maps'
import { useAuthStore } from './auth'

export const useMapStore = defineStore('map', () => {
  const progress = ref(null)
  const loading = ref(false)
  const auth = useAuthStore()
  let pending = null

  const completedLevels = computed(() => progress.value?.completedLevels || 0)
  const totalLevels = computed(() => progress.value?.totalLevels ?? 0)
  const totalXp = computed(() => progress.value?.totalXp ?? 0)
  const currentMap = computed(() => progress.value?.currentMapName || '')

  const overallPct = computed(() =>
    totalLevels.value ? Math.round((completedLevels.value / totalLevels.value) * 100) : 0
  )

  watch(() => auth.token, () => {
    progress.value = null
    pending = null
    loading.value = false
  }, { flush: 'sync' })

  function fetchProgress(force = false) {
    if (!auth.token) return Promise.resolve(null)
    if (pending) return pending
    if (!force && progress.value) return Promise.resolve(progress.value)
    loading.value = true
    const request = mapsApi.progress().then(response => {
      if (pending !== request) return null
      progress.value = response.data.data
      return progress.value
    }).catch(() => null).finally(() => {
      if (pending === request) {
        pending = null
        loading.value = false
      }
    })
    pending = request
    return request
  }

  function refresh() {
    // An in-flight response may predate the submission that triggered the refresh.
    pending = null
    return fetchProgress(true)
  }

  return { progress, loading, completedLevels, totalLevels, totalXp, currentMap, overallPct, fetchProgress, refresh }
})
