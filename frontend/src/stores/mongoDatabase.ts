import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { MongoDatabase, MongoCollection } from '../types/mongo'
import { MongoDatabaseAPI } from '../api/mongo'

export const useMongoDatabaseStore = defineStore('mongoDatabase', () => {
  // State
  const databases = ref<MongoDatabase[]>([])
  const currentDatabase = ref<string | null>(null)
  const collections = ref<MongoCollection[]>([])
  const currentCollection = ref<string | null>(null)
  const isLoading = ref(false)

  // Actions
  async function loadDatabases(profileId: string): Promise<void> {
    isLoading.value = true
    try {
      databases.value = await MongoDatabaseAPI.listDatabases(profileId)
    } finally {
      isLoading.value = false
    }
  }

  async function loadCollections(profileId: string, dbName: string): Promise<void> {
    isLoading.value = true
    try {
      collections.value = await MongoDatabaseAPI.listCollections(profileId, dbName)
    } finally {
      isLoading.value = false
    }
  }

  function selectDatabase(dbName: string): void {
    currentDatabase.value = dbName
    currentCollection.value = null
    collections.value = []
  }

  function selectCollection(collName: string): void {
    currentCollection.value = collName
  }

  async function createCollection(profileId: string, dbName: string, collName: string): Promise<void> {
    await MongoDatabaseAPI.createCollection(profileId, dbName, collName)
    await loadCollections(profileId, dbName)
  }

  async function dropCollection(profileId: string, dbName: string, collName: string): Promise<void> {
    await MongoDatabaseAPI.dropCollection(profileId, dbName, collName)
    if (currentCollection.value === collName) {
      currentCollection.value = null
    }
    await loadCollections(profileId, dbName)
  }

  return {
    // State
    databases,
    currentDatabase,
    collections,
    currentCollection,
    isLoading,
    // Actions
    loadDatabases,
    loadCollections,
    selectDatabase,
    selectCollection,
    createCollection,
    dropCollection,
  }
})
