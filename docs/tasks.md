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
    - [x] 实现 **"资源池实时大屏"**：动态展示当前各池子的 GPU 占用、型号及切分模式。
- [x] **测试验证**：
    - [x] 单元测试：`kubernetes_test.go`（验证 Slicing Mode 识别与 GPU 提取逻辑）。
    - [x] 集成测试：`test_api.sh`（验证 `/api/v1/pools`, `/api/v1/traces` 接口）。
    - [x] 模拟数据：`mock_data.go`（可一键清空并重置演示环境）。

---

## 🟡 第二阶段：池化计费与聚合引擎 (Billing & Consolidation) - [进行中 🚀]

### 后端 - 存储层增强
- [ ] 在 MySQL `life_trace` 表恢复并固化 **聚合指标字段**：
    - `gpu_util_avg` (FLOAT)：会话内 GPU 利用率均值
    - `gpu_util_max` (FLOAT)：会话内 GPU 利用率峰值
    - `mem_used_max` (BIGINT)：会话内显存使用峰值 (MiB)
    - `power_usage_avg` (FLOAT)：会话内平均功耗 (W)
    - `cost_amount` (DECIMAL)：会话计费金额 (元)
- [ ] 在 MySQL `life_trace` 表新增 **业务归属字段**（支持下钻追溯）：
    - `team_label` (VARCHAR)：从 Pod Label `app.kubernetes.io/team` 提取
    - `project_label` (VARCHAR)：从 Pod Label `app.kubernetes.io/project` 提取
    - 需同步更新 **Phase 1 Collector** 的 Pod 入库逻辑，在感知阶段就写入这两个字段。
- [ ] 实现 `GetPendingMetricsTraces` 方法：查询 `end_time IS NOT NULL AND gpu_util_avg = 0` 的待缝合记录。
- [ ] 实现 `UpdateLifeTraceMetrics` 方法：将缝合结果回填至对应记录。
- [ ] 新建 `pool_pricing` 表（池子单价 + Slicing Mode 权重 JSON）。
- [ ] 新建 `daily_billing_snapshot` 表（日级账单聚合快照）。

### 后端 - 指标缝合引擎 (Metrics Stitching)
- [ ] 开发 `backend/internal/aggregator/stitcher.go`：
    - [ ] 定时扫描待缝合记录（建议 10 分钟间隔）。
    - [ ] 以 `start_time ~ end_time` 为区间，向 Prometheus 发起 `range_query`，查询 `DCGM_FI_DEV_GPU_UTIL`、`DCGM_FI_DEV_FB_USED`、`DCGM_FI_DEV_POWER_USAGE`。
    - [ ] 计算各指标的 Avg 与 Max。
    - [ ] 调用 `UpdateLifeTraceMetrics` 持久化回填。

### 后端 - 定价引擎 (Pricing Engine)
- [ ] 开发 `backend/internal/aggregator/pricing.go`：
    - [ ] 实现 `LoadPoolPricing(poolID string) (PoolPricing, error)` 从 MySQL 读取定价配置。
    - [ ] 实现 `CalculateCost(trace LifeTrace, pricing PoolPricing) float64` 的计费公式：
        ```
        cost = (end_time - start_time).Hours() × base_price × slicing_weights[slicing_mode]
        ```
    - [ ] 在缝合完成后自动写入 `life_trace.cost_amount`。

### 后端 - 1d 聚合引擎 (Daily Aggregation)
- [ ] 开发 `backend/internal/aggregator/daily.go`：
    - [ ] 每日凌晨 01:00 触发（使用 Go `cron` 或轮询实现）。
    - [ ] 按 `pool_id + namespace` 聚合当日所有 `life_trace` 记录。
    - [ ] 计算 P95 GPU 利用率、显存峰值、成本合计、会话数。
    - [ ] Upsert 至 `daily_billing_snapshot` 表。

### 后端 - 计费 API (Billing API)
- [ ] 在 `backend/cmd/main.go` 注册 `/api/v2/` 路由组：
    - [ ] `GET /api/v2/billing/daily` - 日级账单汇总（支持按 Pool、Namespace 过滤）。
    - [ ] `GET /api/v2/billing/sessions` - Pod 会话级账单明细。
    - [ ] `GET /api/v2/pricing` - 查询资源池定价配置。
    - [ ] `PUT /api/v2/pricing/:pool_id` - 管理员更新定价规则。

### 前端 - 成本中心与财务报表
- [ ] 实现 **"成本分摊看板"** (`/billing`)：
    - [ ] 按业务组（Namespace）展示日/周/月度成本曲线。
    - [ ] 按资源池维度展示 GPU 利用率与成本趋势对比。
- [ ] 开发 **"账单明细列表"**：支持按时间段、Namespace、Pool 筛选 Pod 级账单。
- [ ] 开发 **"管理员调价后台"** (`/admin/pricing`)：图形化配置各池子单价和切片权重。

### 测试与验证
- [ ] 单元测试：计费公式、聚合逻辑的核心算法测试。
- [ ] 集成测试：`mock_step4_stitch.go` 验证端到端缝合与计费流程。
- [ ] API 测试：`test_api.sh` 扩展 v2 接口验证。

---

## 🔵 第三阶段：AI 专家诊断与摘要算法 (LLM Insights)
- [ ] **后端：数据脱水与 LLM 集成**：
    - [ ] 编写统计降维逻辑，将历史监控数据压缩为适配 LLM 的"特征摘要"。
    - [ ] 基于摘要数据驱动 Gemini/GPT 生成带根因分析的治理报告。
- [ ] **前端：AI 诊断报告交互**：
    - [ ] 实现 **"AI 专家建议卡片"**：展示治理根因、预期收益 (ROI) 及优化 Patch 预览。

## 🔴 第四阶段：可视化看板与闭环治理 (Governance & ROI)
- [ ] **后端：治理执行器 (Pilot)**：
    - [ ] 对接 K8s API 执行资源回收、规格微调与跨池迁移。
- [ ] **前端：治理执行中心与审批流**：
    - [ ] 实现 **"一键治理"控制台**：支持对 AI 建议进行人工确认/执行。
    - [ ] 完善 **"治理 ROI 闭环看板"**：量化展示治理后的实际成本节省数据。

---
> [!TIP]
> 任务状态更新规则：`[ ]` 待启动, `[/]` 进行中, `[x]` 已完成。
