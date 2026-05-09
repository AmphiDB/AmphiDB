import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { ConnectionProfile } from '../types/api';

export const useConnectionStore = defineStore('connection', () => {
  // 当前活跃的连接列表（支持多连接）
  const activeConnections = ref<Map<string, ConnectionProfile>>(new Map());
  const profiles = ref<ConnectionProfile[]>([]);

  // 当前选中的连接（用于工作台等）
  const currentConnection = ref<ConnectionProfile | null>(null);

  // 是否有任何连接
  const isConnected = computed(() => activeConnections.value.size > 0);

  // 获取所有活跃连接列表
  const activeConnectionList = computed(() => Array.from(activeConnections.value.values()));

  // 检查某个连接是否活跃
  function isActive(profileId: string): boolean {
    return activeConnections.value.has(profileId);
  }

  // 添加活跃连接
  function addActiveConnection(profile: ConnectionProfile) {
    activeConnections.value.set(profile.id, profile);
    // 如果没有当前连接，自动设置为当前连接
    if (!currentConnection.value) {
      currentConnection.value = profile;
    }
  }

  // 移除活跃连接
  function removeActiveConnection(profileId: string) {
    activeConnections.value.delete(profileId);
    // 如果移除的是当前连接，切换到其他活跃连接
    if (currentConnection.value?.id === profileId) {
      const remaining = Array.from(activeConnections.value.values());
      currentConnection.value = remaining.length > 0 ? remaining[0] : null;
    }
  }

  // 设置当前选中的连接
  function setCurrentConnection(profile: ConnectionProfile | null) {
    currentConnection.value = profile;
  }

  function setProfiles(newProfiles: ConnectionProfile[]) {
    profiles.value = newProfiles;
  }

  function addProfile(profile: ConnectionProfile) {
    profiles.value.push(profile);
  }

  function updateProfile(id: string, profile: ConnectionProfile) {
    const index = profiles.value.findIndex(p => p.id === id);
    if (index !== -1) {
      profiles.value[index] = profile;
    }
    // 同时更新活跃连接
    if (activeConnections.value.has(id)) {
      activeConnections.value.set(id, profile);
      if (currentConnection.value?.id === id) {
        currentConnection.value = profile;
      }
    }
  }

  function removeProfile(id: string) {
    profiles.value = profiles.value.filter(p => p.id !== id);
    // 同时移除活跃连接
    removeActiveConnection(id);
  }

  return {
    currentConnection,
    isConnected,
    profiles,
    activeConnections,
    activeConnectionList,
    isActive,
    addActiveConnection,
    removeActiveConnection,
    setCurrentConnection,
    setProfiles,
    addProfile,
    updateProfile,
    removeProfile,
  };
});
