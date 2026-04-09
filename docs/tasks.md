# AIPower-Efficiency-Pilot 开发任务清单 (Detailed Roadmap Tasks)

本文档实时记录项目的开发进度与核心任务指标，确保"感知 -> 计费 -> AI分析 -> 治理"业务闭环的顺利落地。

---

## 🟢 第一阶段：全维感知与资源池打标 (Perception & Identity) - [已完结 ✅]
- [x] **后端：池化采集器 (Pool-Collector) 开发**：
    - [x] 基于 `client-go` 实现 Pod 与资源池 (Pool-ID) 的动态自动关联。
    - [x] 对接 Prometheus 实现 5 分钟级 GPU 实时指标探测（显存、算力、特性）。
    - [x] **MySQL Life-Trace 审计存根实现 (GORM)**：记录 Pod 生命周期（起止时间、所属池、切片模式）。
    - [x] **统一配置管理接入 (Viper)**：支持 YAML 与环境变量双模式。
- [x] **后端：实时状态缓冲区**：
    - [x] 配置 Redis 存储 5 分钟级能效快照，支撑 Dashboard 实时展示。
- [x] **前端：基础架构与实时感知看板**：
    - [x] 初始化 Next.js + Tailwind + ShadcnUI 环境。
    - [x] 实现 **"资源池实时大屏"**：动态展示当前各池子的 GPU 占用、型号及切分模式 (美观度优化完成)。
- [x] **测试验证**：
    - [x] 单元测试：`kubernetes_test.go`（验证 Slicing Mode 识别与 GPU 提取逻辑）。
    - [x] 集成测试：`test_api.sh`（验证 `/api/v1/pools`, `/api/v1/traces` 接口）。
    - [x] 模拟数据：`mock_data.go`（可一键清空并重置演示环境）。

---

## 🟢 第二阶段：池化计费与聚合引擎 (Billing & Consolidation) - [已完结 ✅]

### 后端 - 存储层增强
- [x] 在 MySQL `life_trace` 表恢复并固化 **聚合指标字段** (Avg/Max Util, Max Mem, Cost)。
- [x] 在 MySQL `life_trace` 表新增 **业务归属字段** (`team_label`, `project_label`)。
- [x] 实现 `GetPendingMetricsTraces` 与 `UpdateLifeTraceMetrics` 指标回填逻辑。
- [x] 新建 `pool_pricing` 表并实现定价配置持久化。
- [x] 新建 `daily_billing_snapshot` 表并实现预聚合快照存储。

### 后端 - 指标缝合引擎 (Metrics Stitching)
- [x] 开发 `backend/internal/aggregator/stitcher.go`：支持 Prometheus 窗口查询与模拟缝合。
- [x] 在 `main.go` 启动后台 Worker：每 10 分钟自动扫描并缝合结束会话。

### 后端 - 定价引擎 (Pricing Engine)
- [x] 开发 `backend/internal/aggregator/pricing.go`：支持按秒计费与切片模式权重系数。

### 后端 - 1d 聚合引擎 (Daily Aggregation)
- [x] 开发 `backend/internal/aggregator/daily.go`：实现 Pool + Namespace 维度的日级快照生成。
- [x] 在 `main.go` 启动后台 Worker：每日 01:00 自动执行聚合。

### 后端 - 计费 API (Billing API)
- [x] 在 `backend/cmd/main.go` 注册 `/api/v2/` 路由组：
    - [x] `GET /api/v2/billing/daily` - 获取日级账单快照（已按 Pool + Namespace 预聚合）。
    - [x] `GET /api/v2/billing/sessions` - Pod 会话级账单明细（支持多维度过滤）。
    - [x] `GET /api/v2/pricing` - 查询各资源池定价。
    - [x] `PUT /api/v2/pricing/:pool_id` - 管理员更新定价策略。

### 前端 - 成本中心与财务报表
- [x] **对接真实数据**：`/billing` 概览页接入 V2 趋势图与分布图。
- [x] **新增 Pod 效能审计页**：`/billing/sessions` 展示利用率红黑榜、运行时长与功耗。
- [x] **新增业务分摊看板**：`/billing/teams` 实现按 TeamLabel 的成本二次拆分。
- [x] **新增资源池效能量化**：`/billing/pools` 计算单位算力成本 (Unit Cost) 与 ROI 排名。
- [x] **新增管理员调价后台**：`/admin/pricing` 支持在线修改池化计费权重。

### 测试与验证
- [x] 单元测试：计费公式与 P95 算法逻辑。
- [x] 集成测试：`mock_data` 脚本升级，完整模拟 7 天的历史聚合数据。
- [x] UI 验证：修正侧边栏高亮效果与面包屑动态展示。

---

## 🟢 第二阶段补全：资源池资产管理 (Resource Asset Management) - [已完结 ✅]

### 后端 - 资产自动发现
- [x] MySQL 新建 `resource_pool` 元数据表。
- [x] 修改 `K8sCollector.handleNode`：实现发现新 Pool-ID 时自动 Upsert 资产记录。
- [x] 实现 `UpdateResourcePoolMetadata` 方法：支持手动补充业务描述。

### 后端 - 管理 API
- [x] `GET /api/v2/pools`：获取全量已感知的资源池资产列表。
- [x] `PUT /api/v2/pools/:id`：更新资源池业务元数据（别名、场景、备注）。

### 前端 - 资产盘点
- [x] **新增资源池资产清单页** (`/admin/pools/page.tsx`)：
    - [x] 表格化展示已感知的池子。
    - [x] 支持行内编辑业务别名与场景说明。
- [x] **侧边栏导航**：在“配置中心”增加“资源池资产管理”入口。

### 测试增强
- [x] **Mock 脚本升级**：根据命名矩阵预置标准测试池子。

---

## ✅ 第三阶段：AI 专家诊断与摘要算法 (LLM Insights) - [已完成]

### 后端 - LLM 集成层
- [x] **创建 `backend/internal/llm/` 模块目录**
- [x] **实现数据脱水降维 (`summarizer.go`)**：
    - [x] 查询过去 7 天 `daily_billing_snapshot` 数据
    - [x] 按任务维度聚合统计特征（平均利用率、峰值、浪费成本）
    - [x] 识别低利用率 Pod 列表（< 30%）
    - [x] 识别高抖动 Pod 列表（Jitter > 15%）
- [x] **实现 AI 诊断引擎 (`analyzer.go`)**：
    - [x] 封装 LLM API 调用（支持 Gemini/GPT/MiniMax 配置切换）
    - [x] 构建诊断 Prompt 模板
    - [x] 解析 LLM 响应为结构化 `InsightReport`
- [x] **实现报告存储**：
    - [x] 新建 MySQL `insight_reports` 表（含 task_name, namespace, team 字段）
    - [x] 实现报告持久化与查询 API

### 后端 - 管理 API
- [x] **注册 `/api/v3/` 路由组**
- [x] `POST /api/v3/insights/generate` - 手动触发诊断
- [x] `GET /api/v3/insights/reports` - 查询报告列表
- [x] `GET /api/v3/insights/reports/:id` - 查询报告详情
- [x] `PUT /api/v3/insights/reports/:id/status` - 更新报告状态（审批流）

### 前端 - AI 诊断交互
- [x] **新增 AI 诊断页面** (`/insights/page.tsx`)：
    - [x] "生成诊断报告" 按钮
    - [x] 报告列表展示（卡片式）
    - [x] 报告详情展开面板
- [x] **AI 专家建议卡片组件**：
    - [x] 展示治理根因
    - [x] 展示预期 ROI（年度节省/增加金额）
    - [x] 区分降级（节省）vs 稳定性升级（增加）费用显示
- [x] **集成审批流交互**：
    - [x] "批准执行" 按钮
    - [x] "拒绝" 按钮
    - [x] 审批状态实时更新

### 核心优化 - V2 增强
- [x] **任务视角**：诊断对象从"资源池"转为"任务（Pod/PyTorchJob）"
- [x] **团队归属**：报告关联团队（team_label）便于责任追踪
- [x] **费用正负**：降级=节省（正数），稳定性升级=增加（负数）

#### 治理建议增强（V2）- ✅ 已完成
- [x] **任务画像扩展**：
    - [x] TaskType（任务类型：training/inference/dev）
    - [x] Priority（优先级：high/medium/low）
    - [x] HardwareDeps（硬件依赖：NVLink, FP8, MIG, RDMA 等）
    - [x] GPUCount（当前使用 GPU 数量）
- [x] **动作类型**：
    - [x] 降配：只减 GPU 数量，不换池
    - [x] 迁移：只换池，不调规格
    - [x] 降配+迁移：既减 GPU 又换池
- [x] **多建议输出**：一个任务生成 1-3 条建议供选择
- [x] **决策逻辑**：
    - [x] 有硬件依赖 / High优先级 → 只降配不迁移
    - [x] 无硬件依赖 + Low优先级 → 降配+迁移
    - [x] Medium优先级 → 给出多种选择

#### 前端审批增强 - ✅ 已完成
- [x] **单选建议**：从多条建议中选择一条批准
- [x] **最佳建议标识**：自动标识节省最多/增加最少的建议
- [x] **批准记录**：保存批准的建议到数据库（approved_recommendation）
- [x] **状态锁定**：批准/拒绝后不可再选择

### 配置与测试
- [x] **配置管理**：在 `configs/config.yaml` 新增 LLM 配置项
- [x] **单元测试**：
    - [x] `llm/analyzer_test.go` - 降维算法测试
    - [x] `TestGeneratePoolSummary` - 数据脱水测试
    - [x] `TestMismatchDetection` - 任务-资源池不匹配检测测试
    - [x] `TestReportPersistence` - 报告持久化测试
- [x] **集成测试**：`mock_data` 脚本预置任务与资源池不匹配测试数据

## ✅ 第四阶段：治理闭环 (Governance) - [已完成]

### 后端 - 治理执行器
- [x] 治理执行 API
- [x] 降配实现
- [x] 迁移实现
- [x] 降配+迁移实现
- [x] 预览功能
- [x] 执行状态管理
- [x] 操作记录

### 前端 - 治理执行中心
- [x] 待治理任务列表
- [x] 执行预览
- [x] 执行/取消操作
- [x] 执行状态展示

### 前端 - 治理概览
- [x] 执行统计
- [x] 执行记录历史

### 验收标准
- [x] AI 建议可执行
- [x] 执行状态可追踪

---

## 🟢 第二阶段增强：动态切片权重 (Dynamic Slicing Weight) - [已完成]

### 背景
原设计中切片权重是固定值（如 MIG=0.35），无法准确反映 Pod 实际申请的切片量。改进后权重由 `Pod申请量 / 单卡最大量` 动态计算。

### 后端改动

#### 数据模型增强
- [x] `life_trace` 表新增 `slicing_units` (Pod 申请的切片单元数)
- [x] `life_trace` 表新增 `slicing_weight` (动态计算的权重)
- [x] `resource_pool` 表新增 `gpu_vendor` (GPU 厂商)
- [x] `resource_pool` 表新增 `max_slicing_units` (单卡最大切片单元数)

#### 切片检测逻辑
- [x] `parsePoolSlicingConfig()` - 从池子名称解析切片配置
- [x] `countSlicingUnits()` - 统计 Pod 申请的切片单元数
  - Full: 统计 `nvidia.com/gpu` 请求量
  - MIG: 统计 `nvidia.com/mig-*` 请求总和
  - MPS: 从 `CUDA_MPS_ACTIVE_THREAD_PERCENTAGE` 环境变量获取
  - TS: 从 `nvidia.com/gpu-percentage` 注解获取
- [x] `getDefaultMIGUnits()` - GPU 型号 → 最大 MIG 实例数映射表

#### Prometheus 采集增强
- [x] `detectSafeQueryTime()` - 动态检测 Prometheus 延迟，避免数据未就绪
- [x] 使用 `avg_over_time[5m]` 替代即时查询，服务端聚合

#### 计费引擎改动
- [x] `CalculateCost()` 改用动态 `slicing_weight` 参数

#### 后台调度增强
- [x] 修复每日 01:00 聚合逻辑，使用 Timer 精确等待

### 前端改动
- [ ] 暂无前端改动

### 文档更新
- [x] 更新 `perception_and_tagging_logic.md` - 切片配置与权重计算
- [x] 更新 `billing_and_aggregation_logic.md` - 动态权重计费公式
- [x] 更新 `resource_pool_management_logic.md` - 池子命名规范和 MaxSlicingUnits

---
> [!TIP]
> 任务状态更新规则：`[ ]` 待启动, `[/]` 进行中, `[x]` 已完成。
