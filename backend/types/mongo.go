package types

import "time"

// MongoConnectionProfile MongoDB 连接配置
type MongoConnectionProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// 直连模式字段
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	AuthDB   string `json:"authDb"` // 认证数据库，默认 "admin"
	// URI 模式（与直连模式二选一）
	UseURI bool   `json:"useUri"`
	URI    string `json:"uri"` // mongodb:// 或 mongodb+srv://
	// SSH 隧道（与 MySQL 保持一致）
	SSHEnabled  bool   `json:"sshEnabled"`
	SSHHost     string `json:"sshHost,omitempty"`
	SSHPort     int    `json:"sshPort,omitempty"`
	SSHUsername string `json:"sshUsername,omitempty"`
	SSHPassword string `json:"sshPassword,omitempty"`
	SSHKeyPath  string `json:"sshKeyPath,omitempty"`
	// 元数据
	Timeout   int        `json:"timeout"` // 连接超时秒数，默认 10
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// MongoTestResult 连接测试结果
type MongoTestResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ServerVersion string `json:"serverVersion,omitempty"`
	Error         string `json:"error,omitempty"`
}

// MongoDatabase 数据库信息
type MongoDatabase struct {
	Name       string `json:"name"`
	SizeOnDisk int64  `json:"sizeOnDisk"`
}

// MongoCollection 集合信息
type MongoCollection struct {
	Name          string `json:"name"`
	DocumentCount int64  `json:"documentCount"`
}

// MongoQueryParams 文档查询参数
type MongoQueryParams struct {
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Filter     string `json:"filter"`     // JSON 字符串，如 {"status":"active"}
	Sort       string `json:"sort"`       // JSON 字符串，如 {"createdAt":-1}
	Projection string `json:"projection"` // JSON 字符串，如 {"name":1}
	Page       int    `json:"page"`       // 从 1 开始
	PageSize   int    `json:"pageSize"`   // 默认 50
}

// MongoDocumentResult 文档查询结果
type MongoDocumentResult struct {
	Documents []string `json:"documents"` // 每条文档序列化为 JSON 字符串
	Total     int64    `json:"total"`
	Page      int      `json:"page"`
	PageSize  int      `json:"pageSize"`
}

// MongoIndex 索引信息
type MongoIndex struct {
	Name   string         `json:"name"`
	Keys   map[string]int `json:"keys"` // 字段名 → 排序方向（1 或 -1）
	Unique bool           `json:"unique"`
	Sparse bool           `json:"sparse"`
	Type   string         `json:"type"` // single, compound, text, geospatial
}

// MongoIndexSpec 创建索引的规格
type MongoIndexSpec struct {
	Keys   map[string]int `json:"keys"`
	Unique bool           `json:"unique"`
	Sparse bool           `json:"sparse"`
	Name   string         `json:"name,omitempty"`
}

// MongoAggregationResult 聚合查询结果
type MongoAggregationResult struct {
	Documents     []string      `json:"documents"` // 每条结果序列化为 JSON 字符串
	ExecutionTime time.Duration `json:"executionTime"`
	Error         string        `json:"error,omitempty"`
}

// MongoSchemaField 字段分析结果
type MongoSchemaField struct {
	Name      string             `json:"name"`
	Frequency float64            `json:"frequency"` // 出现频率 0.0~1.0
	Types     map[string]float64 `json:"types"`     // 类型名 → 占比
}

// MongoSchemaAnalysis 集合 Schema 分析结果
type MongoSchemaAnalysis struct {
	Collection string             `json:"collection"`
	SampleSize int                `json:"sampleSize"`
	TotalDocs  int64              `json:"totalDocs"`
	Fields     []MongoSchemaField `json:"fields"`
}

// MongoExportParams 导出参数
type MongoExportParams struct {
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Filter     string `json:"filter"` // 空字符串表示导出全部
	Format     string `json:"format"` // "json" 或 "csv"
	FilePath   string `json:"filePath"`
}
