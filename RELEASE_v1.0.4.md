# AmphiDB v1.0.4

This release focuses on AI-assisted SQL authoring, safer large data transfer workflows, and a clearer multi-connection workspace experience.

## Highlights

- Added OpenAI-compatible LLM configuration with encrypted API key storage.
- Added natural-language-to-SQL generation in the SQL editor and table data view.
- Improved LLM SQL prompts to prioritize table and column comments before physical names.
- Added shared SQL completion logic for table names, fields, aliases, SQL functions, and keywords.
- Improved large import/export workflows with background tasks, batching, progress events, and cancellation.
- Fixed stale tabs and schema/data requests leaking across database switches.
- Improved dialog sizing, default window size, typography, and sidebar selection styling.

## Safety

- Natural language SQL generation defaults to read-only `SELECT`/`WITH` statements.
- Generated SQL is parsed and validated server-side before being returned to the UI.
- LLM API keys are stored encrypted and are not returned to the frontend in plain text.
- Large import/export jobs run in cancellable background tasks and avoid loading all rows into memory at once.

## Verification

- `go test ./backend/internal/llm ./backend`
- `npm run build`
- `git diff --check`

Known type-check debt remains in older frontend components when running `npm run build:check`.
