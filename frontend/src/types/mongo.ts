// Type definitions for MongoDB frontend-backend communication

export interface MongoConnectionProfile {
  id: string
  name: string
  host: string
  port: number
  username: string
  password: string
  authDb: string
  useUri: boolean
  uri: string
  sshEnabled: boolean
  sshHost?: string
  sshPort?: number
  sshUsername?: string
  sshPassword?: string
  sshKeyPath?: string
  timeout: number
  createdAt?: string
  updatedAt?: string
}

export interface MongoTestResult {
  success: boolean
  message: string
  serverVersion?: string
  error?: string
}

export interface MongoDatabase {
  name: string
  sizeOnDisk: number
}

export interface MongoCollection {
  name: string
  documentCount: number
}

export interface MongoQueryParams {
  database: string
  collection: string
  filter: string
  sort: string
  projection: string
  page: number
  pageSize: number
}

export interface MongoDocumentResult {
  documents: string[]  // each document serialized as a JSON string
  total: number
  page: number
  pageSize: number
}

export interface MongoIndex {
  name: string
  keys: Record<string, number>
  unique: boolean
  sparse: boolean
  type: string
}

export interface MongoIndexSpec {
  keys: Record<string, number>
  unique: boolean
  sparse: boolean
  name?: string
}

export interface MongoAggregationResult {
  documents: string[]
  executionTime: number
  error?: string
}

export interface MongoSchemaField {
  name: string
  frequency: number
  types: Record<string, number>
}

export interface MongoSchemaAnalysis {
  collection: string
  sampleSize: number
  totalDocs: number
  fields: MongoSchemaField[]
}

export interface MongoExportParams {
  database: string
  collection: string
  filter: string
  format: 'json' | 'csv'
  filePath: string
}
