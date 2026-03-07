# AIPower-Efficiency-Pilot 项目方案 (全栈智能 FinOps 终极版)

## 1. 项目概要 (Executive Summary)
本项目名为 **AIPower-Efficiency-Pilot** (AI 算力效能领航者)，是一款专为 Kubernetes 云原生 AI 架构设计的 **全栈智能 FinOps 效能治理平台**。它通过深度集成 **“带切分标识的资源池 (Pool-Based Slicing)”** 体系，实现 AI 资源的精细画像、动态成本核算及 AI 驱动的效能治理闭环。

## 2. 核心功能点 (Functional Modules)

| 功能模块 | 具体功能点 | 描述 |
| :--- | :--- | :--- |
| **AIPower-Collector** | **池化感知采集** | 实时识别 Pod 所在的资源池（如 `Infer-A100-MIG-Pool`），建立物理与业务的强绑定。 |
| **AIPower-Cost-Engine** | **池化计费引擎** | **核心逻辑：** 屏蔽底层硬件异构，按资源池维度进行定价。基于 `Full/MIG/MPS/TS` 切分模式执行差异化分摊算法。 |
| **AIPower-Analyzer** | **智能效能画像** | 探测任务场景（训/推/研）与池子能力的匹配度。**通过后端统计摘要算法规避 Token 约束**，驱动 **LLM 智能顾问** 生成带根因分析的深度优化报告。 |
| **AIPower-Pilot** | **治理执行中心** | 对接 K8s 执行资源的自动化回收、规格微调及跨池漂移，支持 AI 生成的 Patch 预览。 |
| **AIPower-Dashboard** | **可视化效能看板** | 展示各池子的 ROI 曲线、成本分摊报表、管理员调价后台及 AI 专家建议报告。 |

## 3. 总体架构与业务逻辑 (Detailed Architecture & Logic)

### 3.1 架构拓扑 (Topology Layout)
```text
+-----------------------+      +-----------------------+      +-----------------------+
|   数据采集层 (Eyes)    |      |   分析计费层 (Brain)   |      |   展示执行层 (Face)   |
+-----------+-----------+      +-----------+-----------+      +-----------+-----------+
            |                              |                              |
[K8s Pods / GPU Nodes]         [AIPower-Analyzer]             [AIPower-Dashboard]
      |      |                         |      |                         |
      |      +----(资源池映射)----+----> [ LLM Agent ] <----(专家建议)----+
      |                                |      |                               |
[Prometheus] <---(指标采集)---> [AIPower-Collector] <---(治理建议)---> [AIPower-Pilot]
                                       |      |                               |
                                       +------|-------------------------------+
                                              |
            +---------------------+-----------+-----------+
            |                     |                       |
      [ Redis 实时快照 ]     [ MySQL 账单/建议 ]      [ K8s API 调优 ]
```

### 3.2 架构分层详解 (Architectural Layering)
*   **采集层 (The Eyes):** 采用 `client-go` 机制实时捕获集群 Pod 事件。核心任务是执行 **“池化打标”**，根据调度路径识别其所在的 **Pool-ID**，并将硬件指标与业务标签关联。
*   **分析与计费层 (The Brain):**
    *   **Cost Engine:** 基于 **“池化定价模型”**，根据资源池的 `Slicing Mode` 动态应用分摊权重。
    *   **Analyzer:** 执行 **“数据脱水与摘要策略”**。后端算法首先将海量 Prometheus 指标压缩为统计摘要，以**适配 LLM 的输入 Token 约束**；随后驱动 **LLM 智能顾问** 基于这些特征生成带有根因分析的 Markdown 格式优化报告。
*   **展示与执行层 (The Face & Hands):** 
    *   **Dashboard:** 提供精细化的资源池治理大屏与单价配置管理界面。
    *   **Pilot:** 将分析建议转化为 K8s API 调用，实现自动化资源治理。

### 3.3 核心业务逻辑概述 (Business Logic Overview)
1.  **感知 (Trace):** 建立“资源池-切分模式-业务身份”的绑定，解决资源消耗的归属权问题。
2.  **核算 (Bill):** 采用 **“基准价 + 切片权重”** 计费模型。基准价 (`Pool_Base_Price`) 可基于公有云实例动态市场价或自建机房硬件的折旧成本核算得出。
3.  **诊断 (Analyze):** 后端算法负责**“降维压缩数据”**，AI 顾问负责**“输出专家报告”**。结合硬件特性校验，识别任务是否错配了资源池。
4.  **治理 (Govern):** 将建议落地为真实的成本节省，并动态量化治理 ROI。治理执行需结合**审批流**以保障业务连续性。

## 4. 技术栈说明 (Full-Stack Tech Stack)

| 维度 | 选型 | 理由 |
| :--- | :--- | :--- |
| **后端语言** | **Go (Golang) 1.21+** | 云原生标准，高效集成 client-go 与 Prometheus API。 |
| **前端框架** | **Next.js + React + TS** | 全栈展示层，极致的用户交互体验，支持资源池管理大屏。 |
| **核心存储** | **Prometheus + MySQL + Redis** | 分层存储原始指标、池化账单数据与实时能效快照。 |
| **AI 能力** | **Gemini / GPT-4 / Llama3** | 引入大模型，基于精简后的统计特征生成深度专家报告。 |

## 5. 项目计划与迭代路线
具体的开发阶段划分与任务细节，请参考 📅 **[开发路线图 (Development Roadmap)](development_roadmap.md)**。

## 6. 预期收益与产出 (Expected Benefits & Outputs)
- **核心产出：**
    - AIPower-Collector (Go) 采集组件。
    - AIPower-Cost-Engine (Go) 计费引擎。
    - AIPower-Dashboard (Next.js) 可视化平台。
    - 自动化治理 Pilot (Go/K8s API) 执行中心。
- **业务收益：**
    - 资源利用率提升: 预计通过跨池治理，提升集群整体 GPU 利用率 **25% - 40%**。
    - 成本透明度: 实现基于“业务-池子”维度的 100% 成本穿透，为公司节省年度算力账单 **30%** 以上。

---
> [!NOTE]
> 本方案的业务初衷与核心痛点分析请参阅 📜 **[项目价值观说明书](values.md)**。
