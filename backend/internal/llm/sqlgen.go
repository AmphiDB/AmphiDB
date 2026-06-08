package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Config struct {
	Enabled     bool    `json:"enabled"`
	BaseURL     string  `json:"baseUrl"`
	APIKey      string  `json:"apiKey,omitempty"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	TimeoutSec  int     `json:"timeoutSec"`
}

type PublicConfig struct {
	Enabled       bool    `json:"enabled"`
	BaseURL       string  `json:"baseUrl"`
	Model         string  `json:"model"`
	Temperature   float64 `json:"temperature"`
	TimeoutSec    int     `json:"timeoutSec"`
	HasAPIKey     bool    `json:"hasApiKey"`
	MaskedAPIKey  string  `json:"maskedApiKey"`
	LastUpdatedAt string  `json:"lastUpdatedAt,omitempty"`
}

type TableContext struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Comment string   `json:"comment,omitempty"`
}

type GenerateSQLRequest struct {
	Prompt       string         `json:"prompt"`
	ProfileID    string         `json:"profileId"`
	Database     string         `json:"database"`
	CurrentTable string         `json:"currentTable,omitempty"`
	Tables       []TableContext `json:"tables"`
	AllowWrite   bool           `json:"allowWrite"`
}

type GenerateSQLResponse struct {
	SQL         string `json:"sql"`
	Raw         string `json:"raw"`
	Model       string `json:"model"`
	Explanation string `json:"explanation,omitempty"`
}

type Client interface {
	Generate(ctx context.Context, cfg Config, systemPrompt string, userPrompt string) (string, error)
}

type OpenAIClient struct{}

func (c OpenAIClient) Generate(ctx context.Context, cfg Config, systemPrompt string, userPrompt string) (string, error) {
	cfg = NormalizeConfig(cfg)
	if cfg.APIKey == "" {
		return "", errors.New("LLM API Key 未配置")
	}
	if cfg.Model == "" {
		return "", errors.New("LLM 模型未配置")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)
	res, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: cfg.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.DeveloperMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Temperature: openai.Float(cfg.Temperature),
		MaxTokens:   openai.Int(800),
	})
	if err != nil {
		return "", err
	}
	if res == nil || len(res.Choices) == 0 {
		return "", errors.New("LLM 未返回内容")
	}
	content := strings.TrimSpace(res.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("LLM 返回内容为空")
	}
	return content, nil
}

type Service struct {
	client Client
}

func NewService(client Client) *Service {
	if client == nil {
		client = OpenAIClient{}
	}
	return &Service{client: client}
}

func NormalizeConfig(cfg Config) Config {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 45
	}
	if cfg.TimeoutSec > 180 {
		cfg.TimeoutSec = 180
	}
	if cfg.Temperature < 0 {
		cfg.Temperature = 0
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.1
	}
	if cfg.Temperature > 1 {
		cfg.Temperature = 1
	}
	return cfg
}

func (s *Service) GenerateSQL(ctx context.Context, cfg Config, req GenerateSQLRequest) (GenerateSQLResponse, error) {
	cfg = NormalizeConfig(cfg)
	if !cfg.Enabled {
		return GenerateSQLResponse{}, errors.New("LLM 未启用")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return GenerateSQLResponse{}, errors.New("请输入白话查询需求")
	}
	if strings.TrimSpace(req.Database) == "" {
		return GenerateSQLResponse{}, errors.New("请先选择数据库")
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	systemPrompt := BuildSQLSystemPrompt(req.AllowWrite)
	userPrompt := BuildSQLUserPrompt(req)
	raw, err := s.client.Generate(callCtx, cfg, systemPrompt, userPrompt)
	if err != nil {
		return GenerateSQLResponse{}, fmt.Errorf("LLM 生成 SQL 失败: %w", err)
	}

	sqlText, explanation, err := ExtractSQL(raw)
	if err != nil {
		return GenerateSQLResponse{}, err
	}
	if err := ValidateGeneratedSQL(sqlText, req.AllowWrite); err != nil {
		return GenerateSQLResponse{}, err
	}

	return GenerateSQLResponse{
		SQL:         sqlText,
		Raw:         raw,
		Model:       cfg.Model,
		Explanation: explanation,
	}, nil
}

func BuildSQLSystemPrompt(allowWrite bool) string {
	mode := "Only generate read-only MySQL SELECT or WITH queries."
	if allowWrite {
		mode = "Generate MySQL SQL. Prefer safe SELECT queries unless the user explicitly asks to modify data."
	}
	return strings.Join([]string{
		"You are a senior MySQL query planner embedded in a database management tool.",
		mode,
		"Return strict JSON only, without markdown fences or extra prose.",
		`JSON shape: {"sql":"...","explanation":"..."}`,
		"Use only databases, tables, columns, and relationships present in the provided schema context.",
		"When matching natural language to schema, first use table comments and column comments, then table names and column names.",
		"Treat comments as business semantics, but never use a commented concept unless it maps to an actual table or column in the schema context.",
		"Do not invent table names, column names, relationships, statuses, enum values, date columns, or business meanings.",
		"Empty sql is required when the request cannot be answered from the schema context; explain the missing table, column, relationship, or ambiguity in explanation.",
		"Generate exactly one SQL statement in sql. Never return multiple statements.",
		"Prefer the current table when the user says this table, current table, records, data, or rows without naming another table.",
		"Use explicit column lists instead of SELECT * unless the user asks for all columns.",
		"Qualify columns with table aliases when more than one table is used.",
		"Use `database`.`table` for table references and backticks around identifiers.",
		"Add LIMIT 100 for potentially large result sets unless the user asks for a specific limit, aggregate-only result, or count.",
		"For fuzzy time phrases such as recently/latest, order by the most appropriate existing time column only if one is present; otherwise return empty sql with explanation.",
		"Never include placeholders such as ? unless the user explicitly asks for a parameterized SQL template.",
	}, "\n")
}

func BuildSQLUserPrompt(req GenerateSQLRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Target database: `%s`\n", req.Database)
	if req.CurrentTable != "" {
		fmt.Fprintf(&b, "Current table priority: %s\n", req.CurrentTable)
	}
	b.WriteString("\nSchema context:\n")
	for _, table := range req.Tables {
		fmt.Fprintf(&b, "Table `%s`.`%s`", req.Database, table.Name)
		if table.Comment != "" {
			fmt.Fprintf(&b, " -- %s", table.Comment)
		}
		b.WriteString("\n")
		if len(table.Columns) > 0 {
			for _, column := range table.Columns {
				fmt.Fprintf(&b, "  - %s\n", column)
			}
		} else {
			b.WriteString("  - No columns were provided for this table.\n")
		}
	}
	b.WriteString("\nGeneration checklist:\n")
	b.WriteString("- If a requested field is not in the schema context, return empty sql.\n")
	b.WriteString("- If joining tables requires a relationship not present in names or columns, return empty sql.\n")
	b.WriteString("- If the request asks for destructive changes while read-only mode is active, return empty sql.\n")
	b.WriteString("- Prefer clear aliases and deterministic ordering for latest/top/recent requests.\n")
	b.WriteString("\nUser request:\n")
	b.WriteString(req.Prompt)
	return b.String()
}

type modelSQLResponse struct {
	SQL         string `json:"sql"`
	Explanation string `json:"explanation"`
}

func ExtractSQL(raw string) (string, string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", "", errors.New("LLM 返回内容为空")
	}

	if parsed, ok := parseJSONSQL(text); ok {
		return sqlFromParsedResponse(parsed)
	}

	text = stripMarkdownFence(text)
	if parsed, ok := parseJSONSQL(text); ok {
		return sqlFromParsedResponse(parsed)
	}

	sqlText := sanitizeSQL(text)
	if sqlText == "" {
		return "", "", errors.New("未能从 LLM 返回内容中提取 SQL")
	}
	return sqlText, "", nil
}

func sqlFromParsedResponse(parsed modelSQLResponse) (string, string, error) {
	explanation := strings.TrimSpace(parsed.Explanation)
	sqlText := sanitizeSQL(parsed.SQL)
	if sqlText == "" {
		if explanation != "" {
			return "", explanation, fmt.Errorf("LLM 未生成 SQL: %s", explanation)
		}
		return "", "", errors.New("LLM 未生成 SQL")
	}
	return sqlText, explanation, nil
}

func parseJSONSQL(text string) (modelSQLResponse, bool) {
	var parsed modelSQLResponse
	if err := json.Unmarshal([]byte(text), &parsed); err == nil && (strings.TrimSpace(parsed.SQL) != "" || strings.TrimSpace(parsed.Explanation) != "") {
		return parsed, true
	}

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err == nil && (strings.TrimSpace(parsed.SQL) != "" || strings.TrimSpace(parsed.Explanation) != "") {
			return parsed, true
		}
	}
	return modelSQLResponse{}, false
}

var fencePattern = regexp.MustCompile("(?s)^```(?:sql|mysql|json)?\\s*(.*?)\\s*```$")

func stripMarkdownFence(text string) string {
	text = strings.TrimSpace(text)
	matches := fencePattern.FindStringSubmatch(text)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return text
}

func sanitizeSQL(text string) string {
	text = stripMarkdownFence(text)
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	text = strings.TrimSpace(strings.Join(cleaned, "\n"))

	if idx := strings.Index(text, ";"); idx >= 0 {
		text = text[:idx+1]
	}
	return strings.TrimSpace(text)
}

func ValidateGeneratedSQL(sqlText string, allowWrite bool) error {
	cleaned := strings.TrimSpace(sqlText)
	if cleaned == "" {
		return errors.New("生成的 SQL 为空")
	}
	if hasMultipleStatements(cleaned) {
		return errors.New("生成结果包含多条 SQL，请拆分后再执行")
	}

	leading := strings.ToUpper(firstKeyword(cleaned))
	if allowWrite {
		return nil
	}
	if leading != "SELECT" && leading != "WITH" {
		return fmt.Errorf("白话查询默认只允许生成 SELECT/WITH，当前生成: %s", leading)
	}
	return nil
}

func hasMultipleStatements(sqlText string) bool {
	trimmed := strings.TrimSpace(sqlText)
	if strings.Count(trimmed, ";") == 0 {
		return false
	}
	withoutLast := strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	return strings.Contains(withoutLast, ";")
}

func firstKeyword(sqlText string) string {
	trimmed := strings.TrimLeft(sqlText, " \t\r\n(")
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "`\"")
}
