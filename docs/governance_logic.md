# AIPower-Efficiency-Pilot 第四阶段：治理闭环 (Governance)

## 1. 阶段概述

第四阶段的核心目标是将 AI 诊断建议转化为实际治理动作，真正实现成本节省。根据 `business_logic.md`，本阶段实现 **"治理闭环 (Governance)"** 功能。

### 业务目标
- 执行已批准的 AI 治理建议
- 实现降配、迁移、降配+迁移三种治理动作
- 追踪治理执行状态
- 核心对象：已批准的任务治理建议

> 注意：本项目为 FinOps 平台，专注于治理建议和执行能力。实际利用率改善和费用节省需要运行后持续监控数据，不在本项目范围内。

---

## 2. 核心业务逻辑

### 2.1 治理动作类型

| 动作类型 | 说明 | 适用场景 |
|---------|------|----------|
| **降配** | 减少 GPU 数量，不换池 | 任务负载下降，需保留当前池特性 |
| **迁移** | 不降规格，换到其他池 | 当前池不满足任务需求 |
| **降配+迁移** | 既减 GPU 又换池 | 降本空间最大化 |

### 2.2 执行流程

```
1. 选择任务：从已批准的报告中选择要执行的任务
2. 预览变更：查看将要执行的变更内容
3. 确认执行：用户确认后系统自动执行
4. 状态追踪：实时更新执行状态
```

### 2.3 执行状态流转

```
pending → executing → completed
              ↓
            failed → retry / rollback
```

---

## 3. 数据模型

### 3.1 治理执行记录 (GovernanceExecution)

```go
type GovernanceExecution struct {
    ID           uint       // 主键
    ReportID     uint       // 关联的 AI 诊断报告 ID
    TaskName     string     // 任务名
    Namespace    string     // 命名空间
    ActionType   string     // 治理动作: downgrade/migrate/downgrade_migrate
    FromPool     string     // 源资源池
    ToPool       string     // 目标资源池
    FromGPU      int        // 原 GPU 数量
    ToGPU        int        // 目标 GPU 数量
    Status       string     // 状态: pending/executing/completed/failed
    ScheduledAt  *time.Time // 计划执行时间
    ExecutedAt   *time.Time // 实际执行时间
    ErrorMsg     string     // 失败原因
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

---

## 4. 执行器逻辑

### 4.1 降配 (Downgrade)

**逻辑**：
1. 获取当前 Pod 的资源配置
2. 计算新的 GPU 数量（根据建议的 to_gpu）
3. Patch Pod spec，降低 resources.limits.nvidia.com/gpu
4. 更新 Pod 标签，记录治理信息

### 4.2 迁移 (Migrate)

**逻辑**：
1. 获取当前 Pod 的配置信息
2. 在目标资源池创建新的 Pod（使用相同配置）
3. 等待新 Pod 就绪
4. 删除原 Pod

### 4.3 降配+迁移 (Downgrade + Migrate)

**逻辑**：
1. 先执行降配（降低 GPU 数量）
2. 再执行迁移（切换到目标资源池）

---

## 5. API 设计

### 5.1 执行治理

```
POST /api/v4/governance/execute

Request:
{
    "report_id": 123,           // AI 诊断报告 ID
    "recommendation": {...},    // 选中的建议
    "execute_now": true         // 是否立即执行
}

Response:
{
    "execution_id": "exec_xxx",
    "status": "executing"
}
```

### 5.2 查询执行列表

```
GET /api/v4/governance/executions?status=pending&limit=10

Response:
{
    "executions": [...],
    "total": 100
}
```

### 5.3 查询执行详情

```
GET /api/v4/governance/executions/:id

Response:
{
    "id": "exec_xxx",
    "task_name": "pytorchjob-cv-train",
    "action_type": "降配+迁移",
    "from_pool": "Train-A100-Full-Pool",
    "to_pool": "Dev-T4-TS-Pool",
    "from_gpu": 8,
    "to_gpu": 4,
    "status": "completed",
    "created_at": "2024-01-01T00:00:00Z"
}
```

### 5.4 取消执行

```
PUT /api/v4/governance/executions/:id/cancel

Response:
{
    "status": "cancelled"
}
```

### 5.5 治理统计

```
GET /api/v4/governance/stats

Response:
{
    "total_executed": 10,
    "pending_count": 5
}
```

---

## 6. 前端页面

### 6.1 治理执行中心 (/governance)

**功能**：
- 展示待治理任务列表（来自已批准的 AI 报告）
- 预览变更内容
- 执行/取消操作按钮
- 执行状态展示

### 6.2 治理概览 (/governance/stats)

**功能**：
- 已执行治理动作统计
- 执行记录历史
- 待执行任务数量

---

## 7. 安全机制

### 7.1 预览确认
执行前展示完整的变更内容，包含：
- 任务信息
- 变更详情（GPU 数量、资源池）

### 7.2 操作留痕
所有治理操作记录到审计日志，包括：
- 操作人
- 操作时间
- 变更内容
- 执行结果

### 7.3 失败处理
- 执行失败时自动记录错误信息
- 支持重试
- 保留回滚能力
