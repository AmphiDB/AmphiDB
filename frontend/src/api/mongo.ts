/**
 * MongoDB API 封装
 *
 * 通过 Wails 生成的绑定调用后端 MongoDB 方法
 */

import { ElMessage } from 'element-plus'
import type {
  MongoConnectionProfile,
  MongoTestResult,
  MongoDatabase,
  MongoCollection,
  MongoQueryParams,
  MongoDocumentResult,
  MongoIndex,
  MongoIndexSpec,
  MongoAggregationResult,
  MongoSchemaAnalysis,
  MongoExportParams,
} from '../types/mongo'

// 调用 Wails 绑定的辅助函数，统一处理错误
const wailsCall = async <T>(fn: () => Promise<T>, methodName: string): Promise<T> => {
  try {
    return await fn()
  } catch (error: any) {
    const msg = error?.message || String(error)
    // 对权限错误显示更友好的提示
    if (msg.includes('Unauthorized') || msg.includes('not authorized')) {
      ElMessage.error(`${methodName} 失败: 当前用户权限不足，请检查数据库账号权限`)
    } else {
      ElMessage.error(`${methodName} 失败: ${msg}`)
    }
    throw error
  }
}

// 直接调用 window.go.backend.App 上的方法
const go = () => (window as any)['go']['backend']['App']

/**
 * MongoDB 连接管理 API
 */
export const MongoConnectionAPI = {
  async createProfile(profile: MongoConnectionProfile): Promise<void> {
    return wailsCall(() => go().MongoCreateProfile(profile), 'MongoCreateProfile')
  },

  async updateProfile(id: string, profile: MongoConnectionProfile): Promise<void> {
    return wailsCall(() => go().MongoUpdateProfile(id, profile), 'MongoUpdateProfile')
  },

  async deleteProfile(id: string): Promise<void> {
    return wailsCall(() => go().MongoDeleteProfile(id), 'MongoDeleteProfile')
  },

  async listProfiles(): Promise<MongoConnectionProfile[]> {
    return wailsCall(() => go().MongoListProfiles(), 'MongoListProfiles')
  },

  async testConnection(profile: MongoConnectionProfile): Promise<MongoTestResult> {
    return wailsCall(() => go().MongoTestConnection(profile), 'MongoTestConnection')
  },

  async connect(profileId: string): Promise<void> {
    return wailsCall(() => go().MongoConnect(profileId), 'MongoConnect')
  },

  async disconnect(profileId: string): Promise<void> {
    return wailsCall(() => go().MongoDisconnect(profileId), 'MongoDisconnect')
  },

  async getConnectionStatus(profileId: string): Promise<string> {
    return wailsCall(() => go().MongoGetConnectionStatus(profileId), 'MongoGetConnectionStatus')
  },
}

/**
 * MongoDB 数据库与集合管理 API
 */
export const MongoDatabaseAPI = {
  async listDatabases(profileId: string): Promise<MongoDatabase[]> {
    return wailsCall(() => go().MongoListDatabases(profileId), 'MongoListDatabases')
  },

  async listCollections(profileId: string, dbName: string): Promise<MongoCollection[]> {
    return wailsCall(() => go().MongoListCollections(profileId, dbName), 'MongoListCollections')
  },

  async createCollection(profileId: string, dbName: string, collName: string): Promise<void> {
    return wailsCall(() => go().MongoCreateCollection(profileId, dbName, collName), 'MongoCreateCollection')
  },

  async dropCollection(profileId: string, dbName: string, collName: string): Promise<void> {
    return wailsCall(() => go().MongoDropCollection(profileId, dbName, collName), 'MongoDropCollection')
  },
}

/**
 * MongoDB 文档管理 API
 */
export const MongoDocumentAPI = {
  async queryDocuments(profileId: string, params: MongoQueryParams): Promise<MongoDocumentResult> {
    return wailsCall(() => go().MongoQueryDocuments(profileId, params), 'MongoQueryDocuments')
  },

  async insertDocument(profileId: string, dbName: string, collName: string, jsonDoc: string): Promise<string> {
    return wailsCall(() => go().MongoInsertDocument(profileId, dbName, collName, jsonDoc), 'MongoInsertDocument')
  },

  async updateDocument(profileId: string, dbName: string, collName: string, docId: string, jsonDoc: string): Promise<void> {
    return wailsCall(() => go().MongoUpdateDocument(profileId, dbName, collName, docId, jsonDoc), 'MongoUpdateDocument')
  },

  async deleteDocuments(profileId: string, dbName: string, collName: string, docIds: string[]): Promise<number> {
    return wailsCall(() => go().MongoDeleteDocuments(profileId, dbName, collName, docIds), 'MongoDeleteDocuments')
  },

  async exportDocuments(profileId: string, params: MongoExportParams): Promise<void> {
    return wailsCall(() => go().MongoExportDocuments(profileId, params), 'MongoExportDocuments')
  },
}

/**
 * MongoDB 索引管理 API
 */
export const MongoIndexAPI = {
  async listIndexes(profileId: string, dbName: string, collName: string): Promise<MongoIndex[]> {
    return wailsCall(() => go().MongoListIndexes(profileId, dbName, collName), 'MongoListIndexes')
  },

  async createIndex(profileId: string, dbName: string, collName: string, spec: MongoIndexSpec): Promise<string> {
    return wailsCall(() => go().MongoCreateIndex(profileId, dbName, collName, spec), 'MongoCreateIndex')
  },

  async dropIndex(profileId: string, dbName: string, collName: string, indexName: string): Promise<void> {
    return wailsCall(() => go().MongoDropIndex(profileId, dbName, collName, indexName), 'MongoDropIndex')
  },
}

/**
 * MongoDB 聚合查询 API
 */
export const MongoQueryAPI = {
  async runAggregation(profileId: string, dbName: string, collName: string, pipelineJSON: string): Promise<MongoAggregationResult> {
    return wailsCall(() => go().MongoRunAggregation(profileId, dbName, collName, pipelineJSON), 'MongoRunAggregation')
  },
}

/**
 * MongoDB Schema 分析 API
 */
export const MongoSchemaAPI = {
  async analyzeSchema(profileId: string, dbName: string, collName: string, sampleSize: number): Promise<MongoSchemaAnalysis> {
    return wailsCall(() => go().MongoAnalyzeSchema(profileId, dbName, collName, sampleSize), 'MongoAnalyzeSchema')
  },
}
