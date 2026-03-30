import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { MongoConnectionProfile, MongoTestResult } from '../types/mongo'
import { MongoConnectionAPI } from '../api/mongo'

export const useMongoConnectionStore = defineStore('mongoConnection', () => {
  // State
  const profiles = ref<MongoConnectionProfile[]>([])
  const currentProfileId = ref<string | null>(null)
  const connectionStatuses = ref<Record<string, string>>({})
  const isLoading = ref(false)

  // Getters
  const currentProfile = computed(() =>
    profiles.value.find(p => p.id === currentProfileId.value) ?? null
  )

  function isConnected(profileId: string): boolean {
    return connectionStatuses.value[profileId] === 'connected'
  }

  // Actions
  async function loadProfiles(): Promise<void> {
    isLoading.value = true
    try {
      profiles.value = await MongoConnectionAPI.listProfiles()
    } finally {
      isLoading.value = false
    }
  }

  async function createProfile(profile: MongoConnectionProfile): Promise<void> {
    await MongoConnectionAPI.createProfile(profile)
    await loadProfiles()
  }

  async function updateProfile(id: string, profile: MongoConnectionProfile): Promise<void> {
    await MongoConnectionAPI.updateProfile(id, profile)
    await loadProfiles()
  }

  async function deleteProfile(id: string): Promise<void> {
    await MongoConnectionAPI.deleteProfile(id)
    if (currentProfileId.value === id) {
      currentProfileId.value = null
    }
    await loadProfiles()
  }

  async function testConnection(profile: MongoConnectionProfile): Promise<MongoTestResult> {
    return MongoConnectionAPI.testConnection(profile)
  }

  async function connect(profileId: string): Promise<void> {
    await MongoConnectionAPI.connect(profileId)
    connectionStatuses.value[profileId] = 'connected'
    currentProfileId.value = profileId
  }

  async function disconnect(profileId: string): Promise<void> {
    await MongoConnectionAPI.disconnect(profileId)
    connectionStatuses.value[profileId] = 'disconnected'
  }

  return {
    // State
    profiles,
    currentProfileId,
    connectionStatuses,
    isLoading,
    // Getters
    currentProfile,
    isConnected,
    // Actions
    loadProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
    testConnection,
    connect,
    disconnect,
  }
})
