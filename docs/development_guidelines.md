# AIPower-Efficiency-Pilot 开发规范文档

本文档旨在统一 AIPower-Efficiency-Pilot 项目的开发标准，提升协作效率，确保代码质量。

## 1. 代码风格规范 (Coding Standards)

### 1.1 后端 (Go)
- **核心准则**：遵循 [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)。
- **包管理**：使用 `go mod` 进行依赖管理。
- **项目结构**：
    - `cmd/`: 应用程序入口，每个子目录代表一个二进制文件。
    - `internal/`: 私有应用程序代码，禁止外部导入。
    - `pkg/`: 可由外部项目使用的公共库。
- **命名规范**：
    - 变量名、函数名使用 `camelCase`（私有）或 `PascalCase`（公有）。
    - 缩写词全大写（如 `URL`, `ID`, `JSON`）。
- **错误处理**：显式执行错误检查，避免使用 `panic`。

### 1.2 前端 (TypeScript / Next.js)
- **框架**：Next.js 14+ (App Router)。
- **样式**：使用 Tailwind CSS 进行原子化样式开发。
- **组件规范**：
    - 使用函数式组件 (`Functional Components`)。
    - 统一使用 `Lucide-React` 作为图标库。
    - 状态管理：优先使用 `React Hooks` (useState, useMemo)，复杂状态使用 `Zustand`。
- **Linting**：遵循 ESLint 默认配置及 Prettier 格式化。

---

## 2. Git 提交规范 (Git Workflow)

### 2.1 提交信息格式 (Conventional Commits)
每次提交必须遵循以下格式：
`<type>(<scope>): <subject>`

- **type** 类型包含：
    - `feat`: 新功能
    - `fix`: 修复 Bug
    - `docs`: 文档更新
    - `style`: 格式调整（不影响代码运行）
    - `refactor`: 重构（非新增功能也非修复 Bug）
    - `perf`: 性能优化
    - `build`: 编译系统或外部依赖变更
    - `ci`: CI 配置变更
- **subject**: 简要描述（中文）。

### 2.2 分支模型
- `main`: 稳定主分支，仅接受从 `develop` 发起的 PR。
- `develop`: 开发主分支。
- `feature/*`: 功能开发分支。
- `hotfix/*`: 紧急修复分支。

---

## 3. API 设计规范 (API Design)

### 3.1 路径风格
使用 RESTful 风格，名词复数表示资源：
- `GET /api/v1/pools`: 获取资源池列表
- `POST /api/v1/pools/{id}/adjust`: 调整资源池规格

### 3.2 响应结构
统一包含 `code`, `message`, `data` 字段：
```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

### 3.3 状态码
- `200 OK`: 请求成功
- `201 Created`: 资源创建成功
- `400 Bad Request`: 请求参数错误
- `401 Unauthorized`: 未授权
- `403 Forbidden`: 权限不足
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误

---

## 4. 目录与命名约定

### 4.1 目录结构
```text
.
├── backend/            # Go 后端工程
│   ├── cmd/            # 入口程序
│   └── internal/       # 业务逻辑
├── frontend/           # Next.js 前端工程
│   └── src/            # 前端源码
├── docs/               # 技术与业务文档
├── deploy/             # 部署配置 (K8s YAMLs, Helm)
└── scripts/            # 自动化脚本
```

### 4.2 文件命名
- Go 文件使用 `snake_case.go`。
- TS/TSX 组件使用 `PascalCase.tsx`。
- 样式及配置文件使用 `lowercase` 或 `kebab-case`。
