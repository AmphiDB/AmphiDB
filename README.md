<p align="center">
	<img src="build/appicon.png" alt="AmphiDB Logo" width="120" />
</p>

# AmphiDB

AmphiDB is a modern, cross-platform desktop database management tool for MySQL and MongoDB, built with Wails, Go, and Vue 3.

Repository: https://github.com/AmphiDB/AmphiDB.git

## Features

### MySQL Data Management

- Browse table data with pagination and filtering
- Insert, update, and delete rows
- Run SQL queries and inspect results

### MySQL Structure Management

- Create and alter databases and tables
- Add, modify, and remove columns
- Manage indexes, primary keys, and constraints

### MongoDB Data Management

- Browse, search, and edit collection documents
- Insert, update, and delete documents
- Run aggregation pipelines

### MongoDB Structure Management

- Create and drop databases/collections
- Manage collection indexes
- Analyze collection schema from sampled documents

### Productivity and Utilities

- Query editors for SQL and MongoDB workflows
- Import/export support for SQL, CSV, JSON, and MongoDB workflows
- Schema comparison and synchronization workflows

## Tech Stack

- Backend: Go, Wails v2
- Frontend: Vue 3, TypeScript, Element Plus
- Local config storage: SQLite

## Requirements

- Go 1.21+
- Node.js 18+
- Wails CLI v2

## Quick Start

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone repository
git clone https://github.com/AmphiDB/AmphiDB.git
cd AmphiDB

# Start development mode
make dev
```

## Build

```bash
# Build for current platform
make build

# Build for all major platforms
make build-all

# Platform-specific builds
make build-windows
make build-macos
make build-linux
```

## Test

```bash
make test
make test-backend
make test-frontend
```

## Documentation

- [Build Guide](build/BUILD.md)
- [Packaging Guide](build/PACKAGING.md)
- [Icon Guide](build/ICONS.md)
- [Release Checklist](build/RELEASE_CHECKLIST.md)
- Chinese README: [README.zh-CN.md](README.zh-CN.md)

## Support the Project

If AmphiDB helps your work, you can support ongoing development:

<p align="center">
	<img src="donation-qr.jpg" alt="Donation QR Code" width="200" />
</p>

## License

MIT License
