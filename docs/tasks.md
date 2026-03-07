# AIPower-Efficiency-Pilot 开发任务清单 (Detailed Roadmap Tasks)

本文档实时记录项目的开发进度与核心任务指标，确保“感知 -> 计费 -> AI分析 -> 治理”业务闭环的顺利落地。

---

## 🟢 第一阶段：全维感知与资源池打标 (Perception & Identity)
- [x] **工程骨架初始化**：建立 Go (Backend) 与 Next.js (Frontend) 基础环境。
- [x] **池化采集器 (Pool-Collector) 开发**：
    - [x] 基于 `client-go` 实现 Pod 与资源池 (Pool-ID) 的动态自动关联。
    - [x] 对接 Prometheus 实现 5 分钟级 GPU 实时指标探测（显存、算力、特性）。
- [x] **实时状态缓冲区**：
    - [x] 配置 Redis 存储 5 分钟级能效快照，支撑 Dashboard 实时渲染。

## 🟡 第二阶段：池化定价与聚合引擎 (Billing & Consolidation)
- [ ] **资源池定价引擎实现**：
    - [ ] 设计 MySQL `pool_pricing` 表（支持 Pool_Base_Price + Slicing_Weight）。
    - [ ] 开发基于池化模式维度的“按秒计费”逻辑。
- [ ] **1d 聚合引擎**：
    - [ ] 开发每日凌晨聚合逻辑，将海量采样点转为“日级能效快照”。
    - [ ] 计算每日 P95 利用率、Max 显存占用及分摊成本。

## 🔵 第三阶段：AI 专家诊断与摘要算法 (LLM Insights)
- [ ] **数据脱水与摘要算法**：
    - [ ] 编写统计降维逻辑，将历史监控数据压缩为适配 LLM 的“特征摘要”。
- [ ] **LLM Agent 系统集成**：
    - [ ] 基于摘要数据驱动 Gemini/GPT 生成带根因分析的治理报告。
- [ ] **硬件特性审计**：
    - [ ] 审计 TF32/NVLink/MIG 等高级特性的真实利用效能。

## 🔴 第四阶段：可视化看板与闭环治理 (Governance & ROI)
- [ ] **Next.js Dashboard 可视化**：
    - [ ] 实现资源池 ROI 看板、成本分布图及 AI 建议展示卡片。
- [ ] **治理执行器 (Pilot)**：
    - [ ] 对接 K8s API 执行资源回收、规格微调与跨池迁移。
- [ ] **闭环治理流程**：
    - [ ] 完善“人机协同”审批流 (Human-in-the-loop)，支持 Dry-run 预览。

---
> [!TIP]
> 任务状态更新规则：`[ ]` 待启动, `[/]` 进行中, `[x]` 已完成。
