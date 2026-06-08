export interface LLMConfig {
  enabled: boolean
  baseUrl: string
  apiKey?: string
  model: string
  temperature: number
  timeoutSec: number
}

export interface PublicLLMConfig {
  enabled: boolean
  baseUrl: string
  model: string
  temperature: number
  timeoutSec: number
  hasApiKey: boolean
  maskedApiKey: string
  lastUpdatedAt?: string
}

export interface SQLTableContext {
  name: string
  columns: string[]
  comment?: string
}

export interface GenerateSQLRequest {
  prompt: string
  profileId: string
  database: string
  currentTable?: string
  tables?: SQLTableContext[]
  allowWrite: boolean
}

export interface GenerateSQLResponse {
  sql: string
  raw: string
  model: string
  explanation?: string
}
