/**
 * DB Monitoring API 封装
 *
 * 通过 Wails 生成的绑定调用后端监控方法
 */

// 直接调用 window.go.backend.App 上的方法
const go = () => (window as any)['go']['backend']['App']

// ─── TypeScript Interfaces ────────────────────────────────────────────────────

export interface ExecutionLogEntry {
  id: number
  timestamp: string
  connectionId: string
  dbType: 'mysql' | 'mongodb'
  database: string
  collection: string
  queryText: string
  executionTime: number
  rowsAffected: number
  success: boolean
}

export interface SlowQueryEntry {
  id: number
  timestamp: string
  connectionId: string
  dbType: 'mysql' | 'mongodb'
  database: string
  collection: string
  queryText: string
  durationMs: number
  rowsAffected: number
  errorMessage: string
}

export interface DataPoint {
  timestamp: string
  // common
  qps: number
  tps: number
  // MySQL
  threadsConnected: number
  threadsRunning: number
  innodbBufHitRate: number    // 0-100 %
  innodbRowLockWaits: number
  innodbBufPoolReads: number
  // MongoDB
  mongoConnections: number
  mongoPageFaults: number
  mongoMemResident: number    // MB
  mongoGlobalLock: number
}

export interface MonitoringSnapshot {
  profileId: string
  dbType: string
  dataPoints: DataPoint[]
}

// ─── API Functions ────────────────────────────────────────────────────────────

export function GetExecutionLog(profileID: string, dbType: string, limit: number): Promise<ExecutionLogEntry[]> {
  return go().GetExecutionLog(profileID, dbType, limit)
}

export function ClearExecutionLog(profileID: string, dbType: string): Promise<void> {
  return go().ClearExecutionLog(profileID, dbType)
}

export function GetMonitoringSnapshot(profileID: string, dbType: string): Promise<MonitoringSnapshot> {
  return go().GetMonitoringSnapshot(profileID, dbType)
}

export function GetSlowQueryLog(profileID: string, limit: number): Promise<SlowQueryEntry[]> {
  return go().GetSlowQueryLog(profileID, limit)
}

export function ClearSlowQueryLog(profileID: string): Promise<void> {
  return go().ClearSlowQueryLog(profileID)
}

export function SetSlowQueryThreshold(profileID: string, thresholdMs: number): Promise<void> {
  return go().SetSlowQueryThreshold(profileID, thresholdMs)
}

export function GetSlowQueryThreshold(profileID: string): Promise<number> {
  return go().GetSlowQueryThreshold(profileID)
}
