import type { GenerateSQLRequest, GenerateSQLResponse, LLMConfig, PublicLLMConfig } from '../types/llm'

const app = () => {
  const api = (window as any)?.go?.backend?.App
  if (!api) {
    throw new Error('Wails 运行时不可用')
  }
  return api
}

export const LLMAPI = {
  getConfig(): Promise<PublicLLMConfig> {
    return app().GetLLMConfig()
  },

  saveConfig(config: LLMConfig): Promise<PublicLLMConfig> {
    return app().SaveLLMConfig(config)
  },

  testConfig(config: LLMConfig): Promise<void> {
    return app().TestLLMConfig(config)
  },

  generateSQL(request: GenerateSQLRequest): Promise<GenerateSQLResponse> {
    return app().GenerateSQLFromNaturalLanguage(request)
  },
}
