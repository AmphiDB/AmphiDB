# Database Manager

一款基于 Wails、Go 和 Vue 3 构建的跨平台桌面数据库管理工具，核心聚焦 MySQL 与 MongoDB。

## 核心亮点

- 在同一客户端内统一管理 MySQL 和 MongoDB
- 数据管理：查询、筛选、新增、编辑、删除记录/文档
- 结构管理：表/集合、字段、索引、约束等结构操作
- 提供 SQL 编辑与常用数据库操作流程支持
- 支持导入导出与结构同步，便于环境迁移与一致性维护

## 功能概览

### MySQL 数据管理

- 表数据浏览、分页与过滤
- 行数据新增、修改、删除
- 自定义 SQL 执行与结果查看

### MySQL 结构管理

- 数据库与数据表创建、修改
- 字段新增、调整、删除
- 主键、索引与常见约束管理

### MongoDB 数据管理

- 集合文档浏览与检索
- 文档新增、编辑、删除
- 按常见条件进行文档过滤查询

### MongoDB 结构管理

- 数据库与集合创建、删除
- 集合索引管理
- 维护项目使用的集合结构定义

## 技术栈

- 后端：Go, Wails v2
- 前端：Vue 3, TypeScript, Element Plus
- 本地配置存储：SQLite

## 环境要求

- Go 1.21 及以上
- Node.js 18 及以上
- Wails CLI v2

## 快速开始

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 克隆仓库
git clone https://github.com/sean09527/mysqlGui.git
cd mysqlGui

# 启动开发模式
make dev
```

## 构建

```bash
# 构建当前平台
make build

# 构建多平台
make build-all

# 按平台构建
make build-windows
make build-macos
make build-linux
```

## 测试

```bash
make test
make test-backend
make test-frontend
```

## 项目结构

```text
backend/    Go 后端 API 与服务
frontend/   Vue 3 前端应用
build/      构建脚本与打包资源
```

## 相关文档

- build/BUILD.md
- build/PACKAGING.md
- build/ICONS.md
- build/RELEASE_CHECKLIST.md

## 许可证

MIT License
