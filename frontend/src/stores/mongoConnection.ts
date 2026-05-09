import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { MongoConnectionProfile, MongoTestResult } from '../types/mongo'
import { MongoConnectionAPI } from '../api/mongo'

export const useMongoConnectionStore = defineStore('mongoConnection', () => {
  // State
  const profiles = ref<MongoConnectionProfile[]>([])
  const activeConnections = ref<Map<string, MongoConnectionProfile>>(new Map())
  const currentProfileId = ref<string | null>(null)
  const connectionStatuses = ref<Record<string, string>>({})
  const isLoading = ref(false)

  // Getters
  const currentProfile = computed(() =>
    profiles.value.find(p => p.id === currentProfileId.value) ?? null
  )

  // 是否有任何连接
  const isConnected = computed(() => activeConnections.value.size > 0)

  // 获取所有活跃连接列表
  const activeConnectionList = computed(() => Array.from(activeConnections.value.values()))

  // 检查某个连接是否活跃
  function isActive(profileId: string): boolean {
    return activeConnections.value.has(profileId)
  }

  // 添加活跃连接
  function addActiveConnection(profile: MongoConnectionProfile) {
    activeConnections.value.set(profile.id, profile)
    // 如果没有当前连接，自动设置为当前连接
    if (!currentProfileId.value) {
      currentProfileId.value = profile.id
    }
  }

  // 移除活跃连接
  function removeActiveConnection(profileId: string) {
    activeConnections.value.delete(profileId)
    // 如果移除的是当前连接，切换到其他活跃连接
    if (currentProfileId.value === profileId) {
      const remaining = Array.from(activeConnections.value.keys())
      currentProfileId.value = remaining.length > 0 ? remaining[0] : null
    }
  }

  // 设置当前选中的连接
  function setCurrentProfile(profileId: string | null) {
    currentProfileId.value = profileId
  }

  function isConnectedProfile(profileId: string): boolean {
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
    // 同时更新活跃连接
    if (activeConnections.value.has(id)) {
      activeConnections.value.set(id, profile)
    }
  }

  async function deleteProfile(id: string): Promise<void> {
    await MongoConnectionAPI.deleteProfile(id)
    removeActiveConnection(id)
    await loadProfiles()
  }

  async function testConnection(profile: MongoConnectionProfile): Promise<MongoTestResult> {
    return MongoConnectionAPI.testConnection(profile)
  }

  async function connect(profileId: string): Promise<void> {
    await MongoConnectionAPI.connect(profileId)
    connectionStatuses.value[profileId] = 'connected'
    // 添加到活跃连接
    const profile = profiles.value.find(p => p.id === profileId)
    if (profile) {
      addActiveConnection(profile)
    }
  }

  async function disconnect(profileId: string): Promise<void> {
    await MongoConnectionAPI.disconnect(profileId)
    connectionStatuses.value[profileId] = 'disconnected'
    // 从活跃连接中移除
    removeActiveConnection(profileId)
  }

  return {
    // State
    profiles,
    activeConnections,
    currentProfileId,
    connectionStatuses,
    isLoading,
    // Getters
    currentProfile,
    isConnected,
    activeConnectionList,
    isActive,
    // Actions
    addActiveConnection,
    removeActiveConnection,
    setCurrentProfile,
    isConnectedProfile,
    loadProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
    testConnection,
    connect,
    disconnect,
  }
})
