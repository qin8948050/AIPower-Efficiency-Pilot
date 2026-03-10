# AIPower-Efficiency-Pilot 第三阶段：AI 专家诊断与摘要算法

## 1. 阶段概述

第三阶段的核心目标是将海量监控数据转化为可执行的 AI 治理建议。根据 `business_logic.md`，本阶段实现 **"跨池效能画像 (Analyze)"** 功能。

### 业务目标
- 将 7 天日快照数据压缩为适配 LLM 的**统计特征摘要**
- 驱动大模型生成带根因分析的**《跨池迁移专家报告》**
- 识别任务场景与资源池的不匹配问题

---

## 2. 核心业务逻辑

### 2.1 数据脱水降维 (Data Compression)

**输入数据**：
- `daily_billing_snapshot` 表：过去 7 天的日级聚合数据
- `life_trace` 表：Pod 会话级指标明细

**输出**：
- 适配 LLM 的 JSON 特征摘要（控制 Token 消耗）

### 2.2 AI 诊断分析算法

| 分析类型 | 触发条件 | 建议动作 |
|---------|---------|---------|
| **降级迁移 (Downgrade)** | Full/MIG 池中 GPU 利用率 < 30% 持续 3 天 | 建议迁移至 MPS/TS 池 |
| **稳定性升级 (Isolation)** | MPS/TS 池中算力抖动 (Jitter) > 15% | 建议迁移至 MIG 硬隔离池 |
| **特性纠偏** | 高端特性 (NVLink/FP8) 利用率 < 10% | 建议回退至通用池 |

### 2.3 输出产出

- **《跨池迁移专家报告》**：Markdown 格式，包含根因分析、预期 ROI、优化 Patch 预览

---

## 3. 技术架构设计

### 3.1 后端模块划分

```
backend/internal/
├── llm/                    # 新增：LLM 集成层
│   ├── summarizer.go       # 数据脱水降维
│   ├── analyzer.go        # AI 诊断引擎
│   └── prompt.go          # Prompt 模板
└── ...
```

### 3.2 API 设计

| 方法 | 路径 | 描述 |
|-----|------|------|
| `POST` | `/api/v3/insights/generate` | 触发 AI 诊断分析 |
| `GET` | `/api/v3/insights/reports` | 获取历史报告列表 |
| `GET` | `/api/v3/insights/reports/:id` | 获取报告详情 |

### 3.3 数据模型

```go
// AI 特征摘要（脱敏后供 LLM 使用）
type InsightSummary struct {
    PoolID           string          `json:"pool_id"`
    TimeRange        string          `json:"time_range"` // e.g., "7d"
    AvgUtilization   float64         `json:"avg_utilization"`
    MaxUtilization   float64         `json:"max_utilization"`
    CostTotal        float64         `json:"cost_total"`
    WasteCost        float64         `json:"waste_cost"` // 低利用率导致的浪费
    PodCount         int             `json:"pod_count"`
    LowUtilPods      []LowUtilPod    `json:"low_util_pods,omitempty"`
    HighJitterPods   []JitterPod     `json:"high_jitter_pods,omitempty"`
}

type LowUtilPod struct {
    PodName      string  `json:"pod_name"`
    Namespace    string  `json:"namespace"`
    AvgUtil      float64 `json:"avg_util"`
    EstWasteCost float64 `json:"est_waste_cost"`
}

type JitterPod struct {
    PodName   string  `json:"pod_name"`
    Namespace string  `json:"namespace"`
    JitterPct float64 `json:"jitter_pct"`
}

// AI 诊断报告
type InsightReport struct {
    ID          string    `json:"id"`
    GeneratedAt time.Time `json:"generated_at"`
    Type        string    `json:"type"` // "downgrade", "isolation", "feature_mismatch"
    Summary     string    `json:"summary"`
    RootCause   string    `json:"root_cause"`
    Actions     []Action  `json:"actions"`
    EstSavings  float64   `json:"est_savings"` // 预期年度节省
    Status      string    `json:"status"`     // "pending", "approved", "rejected"
}

type Action struct {
    Type       string `json:"type"` // "migrate", "scale_down", "pool_change"
    PodName    string `json:"pod_name"`
    FromPool   string `json:"from_pool"`
    ToPool     string `json:"to_pool"`
    PatchJSON  string `json:"patch_json,omitempty"`
}
```

---

## 4. 开发任务清单

### 4.1 后端 - LLM 集成层

- [ ] **创建 `backend/internal/llm/` 模块目录**
- [ ] **实现数据脱水降维 (`summarizer.go`)**：
  - [ ] 查询过去 7 天 `daily_billing_snapshot` 数据
  - [ ] 按池子维度聚合统计特征（平均利用率、峰值、浪费成本）
  - [ ] 识别低利用率 Pod 列表（< 30%）
  - [ ] 识别高抖动 Pod 列表（Jitter > 15%）
- [ ] **实现 AI 诊断引擎 (`analyzer.go`)**：
  - [ ] 封装 LLM API 调用（支持 Gemini/GPT 配置切换）
  - [ ] 构建诊断 Prompt 模板
  - [ ] 解析 LLM 响应为结构化 `InsightReport`
- [ ] **实现报告存储**：
  - [ ] 新建 MySQL `insight_reports` 表
  - [ ] 实现报告持久化与查询 API

### 4.2 后端 - 管理 API

- [ ] **注册 `/api/v3/` 路由组**
- [ ] `POST /api/v3/insights/generate` - 手动触发诊断
- [ ] `GET /api/v3/insights/reports` - 查询报告列表
- [ ] `GET /api/v3/insights/reports/:id` - 查询报告详情
- [ ] `PUT /api/v3/insights/reports/:id/status` - 更新报告状态（审批流）

### 4.3 前端 - AI 诊断交互

- [ ] **新增 AI 诊断页面** (`/insights/page.tsx`)：
  - [ ] "生成诊断报告" 按钮
  - [ ] 报告列表展示（卡片式）
  - [ ] 报告详情展开面板
- [ ] **新增 AI 专家建议卡片组件**：
  - [ ] 展示治理根因
  - [ ] 展示预期 ROI（年度节省金额）
  - [ ] 展示优化 Patch 预览（JSON Diff）
- [ ] **集成审批流交互**：
  - [ ] "一键审批" 按钮
  - [ ] "拒绝" 按钮
  - [ ] 审批状态实时更新

### 4.4 配置与测试

- [ ] **配置管理**：在 `configs/config.yaml` 新增 LLM 配置项
- [ ] **单元测试**：`llm/summarizer_test.go` - 降维算法测试
- [ ] **集成测试**：`mock_data` 脚本预置低利用率/高抖动测试数据

---

## 5. 数据流示意

```text
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  MySQL/Daily    │────>│  Summarizer      │────>│  LLM Analyzer   │
│  Snapshots      │     │  (数据脱水降维)   │     │  (生成诊断建议)  │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                           │
                                                           v
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Frontend       │<────│  Insight Reports │<────│  MySQL/Reports  │
│  /insights      │     │  API              │     │  存储            │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

---

## 6. 依赖项

- **LLM Provider**：Gemini API / OpenAI API（通过配置切换）
- **前端组件**：复用现有 ShadcnUI 组件（Card, Table, Button）
- **数据库**：新增 `insight_reports` 表

---

> 📌 详细实现逻辑可参考：
> - [业务逻辑说明书](business_logic.md) - 阶段三核心算法
> - [计费与聚合逻辑](billing_and_aggregation_logic.md) - 日聚合数据来源
