package llm

import (
	"context"
	"strings"
	"testing"
)

type fakeClient struct {
	raw string
	err error
}

func (c fakeClient) Generate(ctx context.Context, cfg Config, systemPrompt string, userPrompt string) (string, error) {
	return c.raw, c.err
}

func TestExtractSQLFromMarkdownFence(t *testing.T) {
	sqlText, _, err := ExtractSQL("```sql\nSELECT * FROM users LIMIT 20;\n```")
	if err != nil {
		t.Fatalf("ExtractSQL returned error: %v", err)
	}
	if sqlText != "SELECT * FROM users LIMIT 20;" {
		t.Fatalf("unexpected SQL: %q", sqlText)
	}
}

func TestExtractSQLFromJSON(t *testing.T) {
	sqlText, explanation, err := ExtractSQL(`{"sql":"SELECT id FROM users LIMIT 10;","explanation":"按用户取 id"}`)
	if err != nil {
		t.Fatalf("ExtractSQL returned error: %v", err)
	}
	if sqlText != "SELECT id FROM users LIMIT 10;" {
		t.Fatalf("unexpected SQL: %q", sqlText)
	}
	if explanation != "按用户取 id" {
		t.Fatalf("unexpected explanation: %q", explanation)
	}
}

func TestExtractSQLFromJSONEmptySQLReturnsExplanation(t *testing.T) {
	_, _, err := ExtractSQL(`{"sql":"","explanation":"缺少订单表，无法回答"}`)
	if err == nil {
		t.Fatal("expected empty SQL response to return an error")
	}
	if !strings.Contains(err.Error(), "缺少订单表") {
		t.Fatalf("expected explanation in error, got: %v", err)
	}
}

func TestBuildSQLSystemPromptRequiresStrictJSONAndNoGuessing(t *testing.T) {
	prompt := BuildSQLSystemPrompt(false)
	for _, expected := range []string{
		"Return strict JSON",
		`"sql"`,
		"Do not invent",
		"read-only",
		"Empty sql",
		"first use table comments and column comments",
		"then table names and column names",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("system prompt should contain %q:\n%s", expected, prompt)
		}
	}
}

func TestBuildSQLUserPromptHighlightsCurrentTableAndSchemaRules(t *testing.T) {
	prompt := BuildSQLUserPrompt(GenerateSQLRequest{
		Prompt:       "查最近失败的数据",
		Database:     "app",
		CurrentTable: "orders",
		Tables: []TableContext{{
			Name:    "orders",
			Comment: "订单表",
			Columns: []string{"id bigint -- 主键", "status varchar(20) -- 订单状态", "created_at datetime -- 创建时间"},
		}},
	})
	for _, expected := range []string{
		"Current table priority: orders",
		"`app`.`orders` -- 订单表",
		"- id bigint -- 主键",
		"- status varchar(20) -- 订单状态",
		"User request",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("user prompt should contain %q:\n%s", expected, prompt)
		}
	}
}

func TestValidateGeneratedSQLRejectsWriteByDefault(t *testing.T) {
	err := ValidateGeneratedSQL("DELETE FROM users WHERE id = 1;", false)
	if err == nil {
		t.Fatal("expected write SQL to be rejected")
	}
	if !strings.Contains(err.Error(), "只允许生成") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGeneratedSQLRejectsMultipleStatements(t *testing.T) {
	err := ValidateGeneratedSQL("SELECT * FROM users; SELECT * FROM orders;", false)
	if err == nil {
		t.Fatal("expected multiple statements to be rejected")
	}
}

func TestGenerateSQLUsesClientAndReturnsSanitizedSQL(t *testing.T) {
	service := NewService(fakeClient{raw: "```sql\nSELECT id, name FROM users LIMIT 20;\n```"})
	res, err := service.GenerateSQL(context.Background(), Config{
		Enabled: true,
		APIKey:  "test-key",
		Model:   "test-model",
	}, GenerateSQLRequest{
		Prompt:   "看用户列表",
		Database: "app",
		Tables: []TableContext{{
			Name:    "users",
			Columns: []string{"id", "name"},
		}},
	})
	if err != nil {
		t.Fatalf("GenerateSQL returned error: %v", err)
	}
	if res.SQL != "SELECT id, name FROM users LIMIT 20;" {
		t.Fatalf("unexpected SQL: %q", res.SQL)
	}
	if res.Model != "test-model" {
		t.Fatalf("unexpected model: %q", res.Model)
	}
}
