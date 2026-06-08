package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"mygui/backend/internal/llm"
)

const llmSettingsKey = "llm.openai_compatible"

// GetLLMConfig returns the global OpenAI-compatible LLM configuration without
// exposing the API key.
func (a *App) GetLLMConfig() (*llm.PublicConfig, error) {
	cfg, updatedAt, err := a.loadLLMConfig()
	if err != nil {
		return nil, err
	}
	return publicLLMConfig(cfg, updatedAt), nil
}

// SaveLLMConfig stores the global LLM configuration. If APIKey is empty while an
// existing key exists, the previous encrypted key is kept.
func (a *App) SaveLLMConfig(cfg llm.Config) (*llm.PublicConfig, error) {
	existing, _, _ := a.loadLLMConfig()
	cfg = llm.NormalizeConfig(cfg)
	cfg.Enabled = cfg.Enabled

	if cfg.APIKey == "" && existing.APIKey != "" {
		cfg.APIKey = existing.APIKey
	}
	if cfg.Enabled && cfg.APIKey == "" {
		return nil, errors.New("启用 LLM 前请填写 API Key")
	}
	if cfg.APIKey != "" {
		encrypted, err := a.encryptor.Encrypt(cfg.APIKey)
		if err != nil {
			return nil, fmt.Errorf("加密 LLM API Key 失败: %w", err)
		}
		cfg.APIKey = encrypted
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("序列化 LLM 配置失败: %w", err)
	}
	if err := a.configStorage.SaveSettings(llmSettingsKey, string(data)); err != nil {
		return nil, fmt.Errorf("保存 LLM 配置失败: %w", err)
	}

	saved, updatedAt, err := a.loadLLMConfig()
	if err != nil {
		return nil, err
	}
	return publicLLMConfig(saved, updatedAt), nil
}

// TestLLMConfig verifies the provided config by sending a tiny request through
// the configured OpenAI-compatible endpoint.
func (a *App) TestLLMConfig(cfg llm.Config) error {
	if cfg.APIKey == "" {
		existing, _, _ := a.loadLLMConfig()
		cfg.APIKey = existing.APIKey
	}
	cfg.Enabled = true
	service := llm.NewService(nil)
	_, err := service.GenerateSQL(context.Background(), cfg, llm.GenerateSQLRequest{
		Prompt:   "生成一个测试查询",
		Database: "test",
		Tables: []llm.TableContext{{
			Name:    "users",
			Columns: []string{"id"},
		}},
	})
	return err
}

// GenerateSQLFromNaturalLanguage converts a plain-language request into a SQL
// statement using the saved OpenAI-compatible LLM configuration.
func (a *App) GenerateSQLFromNaturalLanguage(req llm.GenerateSQLRequest) (*llm.GenerateSQLResponse, error) {
	cfg, _, err := a.loadLLMConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errors.New("LLM 未启用，请先在右上角配置")
	}

	if len(req.Tables) == 0 {
		tables, err := a.collectSQLSchemaContext(req.ProfileID, req.Database, req.CurrentTable)
		if err != nil {
			return nil, err
		}
		req.Tables = tables
	}

	res, err := a.llmService.GenerateSQL(context.Background(), cfg, req)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (a *App) loadLLMConfig() (llm.Config, string, error) {
	raw, err := a.configStorage.GetSettings(llmSettingsKey)
	if err != nil {
		if strings.Contains(err.Error(), "setting not found") {
			return llm.NormalizeConfig(llm.Config{}), "", nil
		}
		return llm.Config{}, "", fmt.Errorf("读取 LLM 配置失败: %w", err)
	}

	var cfg llm.Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return llm.Config{}, "", fmt.Errorf("解析 LLM 配置失败: %w", err)
	}
	if cfg.APIKey != "" {
		decrypted, err := a.encryptor.Decrypt(cfg.APIKey)
		if err != nil {
			return llm.Config{}, "", fmt.Errorf("解密 LLM API Key 失败: %w", err)
		}
		cfg.APIKey = decrypted
	}
	return llm.NormalizeConfig(cfg), time.Now().Format(time.RFC3339), nil
}

func publicLLMConfig(cfg llm.Config, updatedAt string) *llm.PublicConfig {
	return &llm.PublicConfig{
		Enabled:       cfg.Enabled,
		BaseURL:       cfg.BaseURL,
		Model:         cfg.Model,
		Temperature:   cfg.Temperature,
		TimeoutSec:    cfg.TimeoutSec,
		HasAPIKey:     cfg.APIKey != "",
		MaskedAPIKey:  maskSecret(cfg.APIKey),
		LastUpdatedAt: updatedAt,
	}
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func (a *App) collectSQLSchemaContext(profileID, database, currentTable string) ([]llm.TableContext, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, errors.New("未选择数据库连接")
	}
	if strings.TrimSpace(database) == "" {
		return nil, errors.New("请先选择数据库")
	}

	manager, err := a.getOrCreateSchemaManager(profileID)
	if err != nil {
		return nil, fmt.Errorf("获取结构管理器失败: %w", err)
	}

	tableNames := []string{}
	if currentTable != "" {
		tableNames = append(tableNames, currentTable)
	}

	tables, err := manager.ListTables(database)
	if err != nil {
		return nil, fmt.Errorf("获取表列表失败: %w", err)
	}
	for _, table := range tables {
		if len(tableNames) >= 20 {
			break
		}
		if table.Name == "" || table.Name == currentTable {
			continue
		}
		tableNames = append(tableNames, table.Name)
	}

	contextTables := make([]llm.TableContext, 0, len(tableNames))
	for _, tableName := range tableNames {
		tableSchema, err := manager.GetTableSchema(database, tableName)
		if err != nil {
			continue
		}
		columns := make([]string, 0, len(tableSchema.Columns))
		for _, col := range tableSchema.Columns {
			colText := col.Name
			if col.Type != "" {
				colText += " " + col.Type
			}
			if col.Comment != "" {
				colText += " -- " + col.Comment
			}
			columns = append(columns, colText)
		}
		contextTables = append(contextTables, llm.TableContext{
			Name:    tableName,
			Columns: columns,
			Comment: tableSchema.Comment,
		})
	}
	if len(contextTables) == 0 {
		return nil, errors.New("未能获取可用表结构上下文")
	}
	return contextTables, nil
}
