# AIPower-Efficiency-Pilot 第三阶段 V2：AI 专家诊断与摘要算法

## 1. 阶段概述

第三阶段的核心目标是将海量监控数据转化为可执行的 AI 治理建议。根据 `business_logic.md`，本阶段实现 **"跨池效能画像 (Analyze)"** 功能。

### 业务目标
- 将 7 天日快照数据压缩为适配 LLM 的**统计特征摘要**
- 驱动大模型生成带根因分析的**《任务治理专家报告》**
- 识别任务场景与资源池的不匹配问题
- **核心对象**：任务（Pod/PyTorchJob），而非资源池

---

## 2. 核心业务逻辑

### 2.1 数据脱水降维 (Data Compression)

**输入数据**：
- `daily_billing_snapshot` 表：过去 7 天的日级聚合数据
- `life_trace` 表：Pod 会话级指标明细

**输出**：
- 适配 LLM 的 JSON 特征摘要（控制 Token 消耗）

### 2.2 任务画像 (Task Profile)

任务画像是治理决策的基础，包含以下维度：

```go
type TaskProfile struct {
    TaskType     TaskType     // 任务类型: training, inference, dev
    Priority     Priority     // 优先级: high, medium, low
    HardwareDeps []string     // 硬件依赖: ["nvlink", "fp8", "mig", "rdma"]
    GPUCount     int          // 当前使用 GPU 数量
    MemUsedGB    float64      // 显存使用量 (GB)
}
```

#### 任务类型识别

| 来源 | 推断结果 |
|-----|---------|
| namespace 包含 "train" | TaskType = training |
| namespace 包含 "infer" | TaskType = inference |
| namespace 包含 "dev" 或 "test" | TaskType = dev |

#### 优先级识别

| 来源 | 推断结果 |
|-----|---------|
| team_label 包含 "core" 或 "prod" | Priority = high |
| team_label 包含 "infra" | Priority = medium |
| 其他 | Priority = low |

#### 硬件依赖识别

| 来源 | 推断结果 |
|-----|---------|
| Pod annotation 包含 "nvlink" | HardwareDeps = ["nvlink"] |
| Pod annotation 包含 "fp8" | HardwareDeps = ["fp8"] |
| Pod annotation 包含 "mig" | HardwareDeps = ["mig"] |
| Pod annotation 包含 "rdma" | HardwareDeps = ["rdma"] |

### 2.3 治理动作类型

| 动作类型 | 说明 | 场景 |
|---------|------|------|
| **降配** | 只减 GPU 数量，不换池 | 有硬件依赖 / 高优先级 |
| **迁移** | 只换池，不调规格 | 池子不匹配但资源刚好 |
| **降配+迁移** | 既减 GPU 又换池 | 无硬件依赖，想最大节省 |

### 2.4 AI 诊断分析算法

#### 问题识别

| 问题 | 条件 |
|-----|------|
| 利用率低 | GPU 利用率 < 30% |
| 利用率高 | GPU 利用率 > 90% |
| 抖动高 | Jitter > 15% |

#### 决策流程

```
问题识别：利用率低 / 抖动高

┌────────────────────────────────────────────────────────────┐
│ 1. 任务有硬件依赖？                                       │
│    └─ 是 → 只生成 "降配" 建议（不迁移）                 │
│    └─ 否 → 继续步骤 2                                    │
├────────────────────────────────────────────────────────────┤
│ 2. 任务优先级是 High？                                    │
│    └─ 是 → 只生成 "降配" 建议                           │
│    └─ 否 → 继续步骤 3                                    │
├────────────────────────────────────────────────────────────┤
│ 3. 利用率低 + 无硬件依赖 + Low优先级？                   │
│    └─ 是 → 生成 "降配+迁移" 建议（降级到TS池）         │
├────────────────────────────────────────────────────────────┤
│ 4. 利用率低 + Medium优先级？                              │
│    └─ 是 → 生成 "降配" + "降配+迁移" 建议（二选一）    │
├────────────────────────────────────────────────────────────┤
│ 5. 抖动高 + 无硬件依赖？                                  │
│    └─ 是 → 生成 "迁移" 建议（升级到MIG池）             │
├────────────────────────────────────────────────────────────┤
│ 6. 抖动高 + Medium优先级？                                │
│    └─ 是 → 生成 "迁移" + "降配+迁移" 建议               │
└────────────────────────────────────────────────────────────┘
```

### 2.5 硬件依赖与迁移边界

| 任务特性需求 | 治理建议 |
|-------------|---------|
| 需要 NVLink + FP8 + RDMA | **只降配**（保留高端特性） |
| 需要 NVLink + RDMA | **只降配**（保留NVLink） |
| 需要 FP8 | **只降配**（保留FP8） |
| 需要 MIG | **只升MIG规格**（保留隔离） |
| 无特殊需求 | 可以跨池迁移 |

**核心原则**：硬件依赖决定迁移边界，降配时保留特性。

---

## 3. 输出产出

### 3.1 多建议输出格式

```json
{
  "task_name": "pytorchjob-distributed-train",
  "namespace": "ai-platform",
  "team": "NLP-Group",
  "problem": "GPU利用率低",
  "current_profile": {
    "task_type": "training",
    "priority": "low",
    "hardware_deps": ["nvlink", "fp8"],
    "gpu_count": 8,
    "avg_util": 18.5,
    "current_pool": "Train-H800-Full-Pool"
  },
  "recommendations": [
    {
      "action_type": "降配",
      "from_gpu": 8,
      "to_gpu": 4,
      "from_pool": "Train-H800-Full-Pool",
      "to_pool": "Train-H800-Full-Pool",
      "est_savings": 8500,
      "reason": "保留NVLink+FP8特性，降至4卡仍满足通信需求"
    },
    {
      "action_type": "降配+迁移",
      "from_gpu": 8,
      "to_gpu": 2,
      "from_pool": "Train-H800-Full-Pool",
      "to_pool": "Dev-T4-TS-Pool",
      "est_savings": 18000,
      "reason": "降配并迁移到低成本TS池，节省更多"
    }
  ]
}
```

---

## 4. 技术架构设计

### 4.1 后端模块划分

```
backend/internal/
├── llm/                    # LLM 集成层
│   ├── summarizer.go       # 数据脱水降维 + 任务画像识别
│   ├── analyzer.go        # AI 诊断引擎 + 治理建议生成
│   └── types.go           # 数据类型定义
```

### 4.2 API 设计

| 方法 | 路径 | 描述 |
|-----|------|------|
| `POST` | `/api/v3/insights/generate` | 触发 AI 诊断分析 |
| `GET` | `/api/v3/insights/reports` | 获取历史报告列表 |
| `GET` | `/api/v3/insights/reports/:id` | 获取报告详情 |
| `PUT` | `/api/v3/insights/reports/:id/status` | 更新报告状态（审批流） |

### 4.3 数据模型

```go
// 任务画像
type TaskProfile struct {
    TaskType     TaskType     // 任务类型
    Priority     Priority     // 优先级
    HardwareDeps []string     // 硬件依赖
    GPUCount     int          // GPU 数量
    MemUsedGB    float64      // 显存使用
}

// 治理建议
type Recommendation struct {
    ActionType  string   // 动作类型: "降配", "迁移", "降配+迁移"
    FromGPU     int      // 当前 GPU 数
    ToGPU       int      // 目标 GPU 数
    FromPool    string   // 当前资源池
    ToPool      string   // 目标资源池
    EstSavings  float64  // 预估节省
    Reason      string   // 建议理由
}

// 治理结果
type GovernanceResult struct {
    TaskName          string           // 任务名
    Namespace         string           // 命名空间
    Team              string           // 团队
    Problem           string           // 问题描述
    CurrentProfile    TaskProfile      // 当前画像
    Recommendations   []Recommendation // 建议列表（1-2条）
}
```

---

## 5. 开发任务清单

### 5.1 后端 - LLM 集成层

- [x] **创建 `backend/internal/llm/` 模块目录**
- [x] **实现数据脱水降维 (`summarizer.go`)**
- [x] **实现 AI 诊断引擎 (`analyzer.go`)**
- [x] **实现报告存储**

#### V2 增强 - 待开发

- [ ] **任务画像扩展**：TaskType, Priority, HardwareDeps, GPUCount
- [ ] **动作类型实现**：降配、迁移、降配+迁移
- [ ] **多建议输出**：一个任务生成 1-2 条建议
- [ ] **决策逻辑**：根据硬件依赖和优先级生成建议

### 5.2 后端 - 管理 API

- [x] `POST /api/v3/insights/generate`
- [x] `GET /api/v3/insights/reports`
- [x] `GET /api/v3/insights/reports/:id`
- [x] `PUT /api/v3/insights/reports/:id/status`

### 5.3 前端 - AI 诊断交互

- [x] **新增 AI 诊断页面** (`/insights/page.tsx`)
- [x] **审批流交互**

#### V2 增强 - 待开发

- [ ] **多建议展示**：展示 1-2 个选项供选择
- [ ] **执行治理**：选择建议后执行

---

## 6. 数据流示意

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  MySQL/Daily    │────>│  Summarizer      │────>│  LLM Analyzer   │
│  Snapshots      │     │  (数据脱水降维)   │     │  (生成诊断建议)  │
└─────────────────┘     │  + 任务画像识别   │     └────────┬────────┘
                        └──────────────────┘              │
                                                            v
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Frontend       │<────│  Insight Reports │<────│  MySQL/Reports  │
│  /insights      │     │  API              │     │  存储            │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

---

## 7. 依赖项

- **LLM Provider**：MiniMax API / Gemini API / OpenAI API（通过配置切换）
- **前端组件**：复用现有 ShadcnUI 组件
- **数据库**：使用现有 `insight_reports` 表

---

> 📌 详细实现逻辑可参考：
> - [业务逻辑说明书](business_logic.md) - 阶段三核心算法
> - [计费与聚合逻辑](billing_and_aggregation_logic.md) - 日聚合数据来源
